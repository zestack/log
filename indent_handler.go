package log

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"
)

type IndentHandler struct {
	opts           slog.HandlerOptions
	preformatted   []byte   // data from WithGroup and WithAttrs
	unopenedGroups []string // groups from WithGroup that haven't been opened
	indentLevel    int      // same as number of opened groups so far
	mu             *sync.Mutex
	out            io.Writer
}

func NewIndentHandler(out io.Writer, opts *slog.HandlerOptions) *IndentHandler {
	h := &IndentHandler{
		out: out,
		mu:  &sync.Mutex{},
	}
	if opts != nil {
		h.opts = *opts
	}
	if h.opts.Level == nil {
		h.opts.Level = slog.LevelInfo
	}
	return h
}

func (h *IndentHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return level >= h.opts.Level.Level()
}

func (h *IndentHandler) WithGroup(name string) slog.Handler {
	if name == "" {
		return h
	}
	h2 := *h
	// Add an unopened group to h2 without modifying h.
	h2.unopenedGroups = make([]string, len(h.unopenedGroups)+1)
	copy(h2.unopenedGroups, h.unopenedGroups)
	h2.unopenedGroups[len(h2.unopenedGroups)-1] = name
	return &h2
}

func (h *IndentHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	if len(attrs) == 0 {
		return h
	}
	h2 := *h
	// Force an append to copy the underlying array.
	pre := slices.Clip(h.preformatted)
	// Add all groups from WithGroup that haven't already been added.
	h2.preformatted = h2.appendUnopenedGroups(pre, h2.indentLevel)
	// Each of those groups increased the indent level by 1.
	h2.indentLevel += len(h2.unopenedGroups)
	// Now all groups have been opened.
	h2.unopenedGroups = nil
	// Pre-format the attributes.
	for _, a := range attrs {
		h2.preformatted = h2.appendAttr(h2.preformatted, a, h2.indentLevel)
	}
	return &h2
}

func (h *IndentHandler) appendUnopenedGroups(buf []byte, indentLevel int) []byte {
	for _, g := range h.unopenedGroups {
		buf = fmt.Appendf(buf, "%*s%s:\n", indentLevel*4, "", g)
		indentLevel++
	}
	return buf
}

func (h *IndentHandler) Handle(ctx context.Context, r slog.Record) error {
	bufp := allocBuf()
	buf := *bufp
	defer func() {
		*bufp = buf
		freeBuf(bufp)
	}()
	if !r.Time.IsZero() {
		buf = h.appendAttr(buf, slog.Time(slog.TimeKey, r.Time), 0)
	}
	buf = h.appendAttr(buf, slog.Any(slog.LevelKey, r.Level), 0)
	if h.opts.AddSource {
		fs := runtime.CallersFrames([]uintptr{r.PC})
		f, _ := fs.Next()
		// Optimize to minimize allocation.
		srcbufp := allocBuf()
		defer freeBuf(srcbufp)
		*srcbufp = append(*srcbufp, f.File...)
		*srcbufp = append(*srcbufp, ':')
		*srcbufp = strconv.AppendInt(*srcbufp, int64(f.Line), 10)
		buf = h.appendAttr(buf, slog.String(slog.SourceKey, string(*srcbufp)), 0)
	}

	buf = h.appendAttr(buf, slog.String(slog.MessageKey, r.Message), 0)
	// Insert preformatted attributes just after built-in ones.
	buf = append(buf, h.preformatted...)
	if r.NumAttrs() > 0 {
		buf = h.appendUnopenedGroups(buf, h.indentLevel)
		r.Attrs(func(a slog.Attr) bool {
			buf = h.appendAttr(buf, a, h.indentLevel+len(h.unopenedGroups))
			return true
		})
	}
	buf = append(buf, "---\n"...)
	h.mu.Lock()
	defer h.mu.Unlock()
	_, err := h.out.Write(buf)
	return err
}

func (h *IndentHandler) appendAttr(buf []byte, a slog.Attr, indentLevel int) []byte {
	// Resolve the Attr's value before doing anything else.
	a.Value = a.Value.Resolve()
	if rep := h.opts.ReplaceAttr; rep != nil && a.Value.Kind() != slog.KindGroup {
		var gs []string
		if h.unopenedGroups != nil {
			gs = h.unopenedGroups[:]
		}
		// a.Value is resolved before calling ReplaceAttr, so the user doesn't have to.
		a = rep(gs, a)
		// The ReplaceAttr function may return an unresolved Attr.
		a.Value = a.Value.Resolve()
	}
	// Ignore empty Attrs.
	if a.Equal(slog.Attr{}) {
		return buf
	}
	if a.Value.Kind() != slog.KindGroup {
		// key
		buf = append(buf, a.Key...)
		buf = append(buf, ": "...)
	}
	switch a.Key {
	case slog.MessageKey:
		// message
		msgbufp := allocBuf()
		defer freeBuf(msgbufp)
		var lines int
		var indent []byte
		msg := a.Value.String()
		for {
			if lines == 1 {
				indent = fmt.Appendf(indent, "%*s", (indentLevel+1)*4, "")
				*msgbufp = append(indent[:], *msgbufp...)
				buf = append(buf, ">-\n"...)
			}
			*msgbufp = append(*msgbufp, indent...)
			index := strings.IndexByte(msg, '\n')
			if index == -1 {
				*msgbufp = append(*msgbufp, strings.TrimSuffix(msg, " \r\n")...)
				*msgbufp = append(*msgbufp, '\n')
				break
			} else {
				*msgbufp = append(*msgbufp, msg[:index]...)
				*msgbufp = append(*msgbufp, '\n')
				msg = msg[index+1:]
			}
			lines++
		}
		buf = append(buf, *msgbufp...)
	case slog.LevelKey:
		buf = append(buf, levelToString(a.Value.Any().(slog.Level))...)
		buf = append(buf, '\n')
	case slog.SourceKey:
		buf = append(buf, a.Value.String()...)
		buf = append(buf, '\n')
	default:
		// Indent 4 spaces per level.
		buf = fmt.Appendf(buf, "%*s", indentLevel*4, "")
		switch a.Value.Kind() {
		case slog.KindString:
			// Quote string values, to make them easy to parse.
			buf = strconv.AppendQuote(buf, a.Value.String())
			buf = append(buf, '\n')
		case slog.KindTime:
			// Write times in a standard way, without the monotonic time.
			buf = a.Value.Time().AppendFormat(buf, time.RFC3339Nano)
			buf = append(buf, '\n')
		case slog.KindGroup:
			attrs := a.Value.Group()
			// Ignore empty groups.
			if len(attrs) == 0 {
				return buf
			}
			// If the key is non-empty, write it out and indent the rest of the attrs.
			// Otherwise, inline the attrs.
			if a.Key != "" {
				buf = fmt.Appendf(buf, "%s:\n", a.Key)
				indentLevel++
			}
			for _, ga := range attrs {
				buf = h.appendAttr(buf, ga, indentLevel)
			}
		default:
			buf = append(buf, a.Value.String()...)
			buf = append(buf, '\n')
		}
	}
	return buf
}
