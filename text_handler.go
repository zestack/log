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
	"zestack.dev/color"
)

type TextHandler struct {
	opts         slog.HandlerOptions
	preformatted []byte   // data from WithGroup and WithAttrs
	groups       []string // all groups started from WithGroup
	mu           *sync.Mutex
	out          color.Writer
}

func NewTextHandler(out io.Writer, opts *slog.HandlerOptions) *TextHandler {
	w, ok := out.(color.Writer)
	if !ok {
		w = color.NewWriter(out)
	}
	h := &TextHandler{out: w, mu: &sync.Mutex{}}
	if opts != nil {
		h.opts = *opts
	}
	if h.opts.Level == nil {
		h.opts.Level = slog.LevelInfo
	}
	return h
}

func (h *TextHandler) clone() TextHandler {
	return TextHandler{
		opts:         h.opts,
		preformatted: h.preformatted[:],
		groups:       h.groups[:],
		mu:           h.mu,
		out:          h.out,
	}
}

func (h *TextHandler) Enabled(_ context.Context, level slog.Level) bool {
	minLevel := slog.LevelInfo
	if h.opts.Level != nil {
		minLevel = h.opts.Level.Level()
	}
	return level >= minLevel
}

func (h *TextHandler) WithGroup(name string) slog.Handler {
	if name == "" {
		return h
	}
	h2 := h.clone()
	// Add an unopened group to h2 without modifying h.
	h2.groups = make([]string, len(h.groups)+1)
	copy(h2.groups, h.groups)
	h2.groups[len(h2.groups)-1] = name
	return &h2
}

func (h *TextHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	if len(attrs) == 0 {
		return h
	}
	h2 := *h
	// Force an append to copy the underlying array.
	h2.preformatted = slices.Clip(h.preformatted)
	h2.groups = slices.Clip(h.groups)
	// Pre-format the attributes.
	for _, a := range attrs {
		h2.preformatted = h2.appendAttr(h2.preformatted, a)
	}
	return &h2
}

func (h *TextHandler) Handle(_ context.Context, r slog.Record) error {
	bufp := allocBuf()
	buf := *bufp
	defer func() {
		*bufp = buf
		freeBuf(bufp)
	}()
	if !r.Time.IsZero() {
		buf = h.appendAttr(buf, slog.Time(slog.TimeKey, r.Time))
	}
	buf = h.appendAttr(buf, slog.Any(slog.LevelKey, r.Level))
	buf = h.appendAttr(buf, slog.String(slog.MessageKey, r.Message))
	if h.opts.AddSource {
		fs := runtime.CallersFrames([]uintptr{r.PC})
		f, _ := fs.Next()
		// Optimize to minimize allocation.
		srcbufp := allocBuf()
		defer freeBuf(srcbufp)
		*srcbufp = append(*srcbufp, f.File...)
		*srcbufp = append(*srcbufp, ':')
		*srcbufp = strconv.AppendInt(*srcbufp, int64(f.Line), 10)
		if strings.Contains(r.Message, "\n") {
			buf = append(buf, ' ')
		}
		buf = h.appendAttr(buf, slog.String(slog.SourceKey, string(*srcbufp)))
	}
	if h.opts.AddSource && strings.Contains(r.Message, "\n") {
		buf = append(buf, "\n  "...)
	}
	buf = append(buf, sDim...)
	// Insert preformatted attributes just after built-in ones.
	buf = append(buf, h.preformatted...)
	if r.NumAttrs() > 0 {
		r.Attrs(func(a slog.Attr) bool {
			buf = h.appendAttr(buf, a)
			return true
		})
	}
	buf = append(buf, cReset...)
	buf = append(buf, "\n"...)
	h.mu.Lock()
	defer h.mu.Unlock()
	_, err := h.out.Write(buf)
	return err
}

var (
	cHour    = color.New(color.FgBlue)
	cYear    = color.New(color.FgMagenta)
	cDim     = color.New(color.FgHiBlack)
	sDefault = color.Bytes(color.FgHiWhite)
	sDim     = color.Bytes(color.FgHiBlack)
	cReset   = color.Bytes(color.Reset)
)

func (h *TextHandler) appendAttr(buf []byte, a slog.Attr) []byte {
	// Resolve the Attr's value before doing anything else.
	a.Value = a.Value.Resolve()
	if rep := h.opts.ReplaceAttr; rep != nil && a.Value.Kind() != slog.KindGroup {
		var gs []string
		if h.groups != nil {
			gs = h.groups[:]
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
	switch a.Key {
	case slog.TimeKey:
		ts := strings.SplitN(a.Value.Time().Format(time.DateTime), " ", 2)
		buf = fmt.Appendf(buf, "%s %s", cYear.Wrap(ts[0]), cHour.Wrap(ts[1]))
		buf = append(buf, ' ')
		return buf
	case slog.LevelKey:
		level, prepend := levelToColor(a.Value.Any().(slog.Level))
		buf = fmt.Appendf(buf, "%s %s%s %s", cDim.Wrap("|"), prepend, level, cDim.Wrap("|"))
		buf = append(buf, ' ')
		return buf
	case slog.MessageKey:
		msgbufp := allocBuf()
		defer freeBuf(msgbufp)
		var prepend []byte
		var lines int
		msg := a.Value.String()
		buf = append(buf, sDefault...)
		for {
			if lines == 1 {
				buf = fmt.Appendf(buf, "%s\n", cDim.Wrap("↲"))
				prepend = append(append(sDim, []byte("  > ")...), cReset...)
				*msgbufp = append(prepend, *msgbufp...)
			}
			*msgbufp = append(*msgbufp, prepend...)
			index := strings.IndexByte(msg, '\n')
			if index == -1 {
				if lines > 1 {
					msg = strings.TrimSpace(msg)
				}
				*msgbufp = append(*msgbufp, msg...)
				if lines > 1 {
					*msgbufp = append(*msgbufp, '\n')
				} else {
					*msgbufp = append(*msgbufp, ' ')
				}
				break
			} else {
				*msgbufp = append(*msgbufp, strings.TrimSpace(msg[:index])...)
				*msgbufp = append(*msgbufp, '\n')
				msg = msg[index+1:]
			}
			lines++
		}
		buf = append(buf, *msgbufp...)
		buf = append(buf, cReset...)
		return buf
	case slog.SourceKey:
		buf = append(buf, cDim.Wrap(a.Key+"=\"").Bytes()...)
		buf = append(buf, color.Namespace(a.Value.String()).Bytes()...)
		buf = append(buf, cDim.Wrap("\"").Bytes()...)
		buf = append(buf, ' ')
		return buf
	default:
		if a.Value.Kind() != slog.KindGroup {
			for _, g := range h.groups {
				buf = fmt.Appendf(buf, "%s.", g)
			}
		}
	}
	switch a.Value.Kind() {
	case slog.KindString:
		// Quote string values, to make them easy to parse.
		buf = append(buf, a.Key...)
		buf = append(buf, "="...)
		buf = strconv.AppendQuote(buf, a.Value.String())
		buf = append(buf, ' ')
	case slog.KindTime:
		// Write times in a standard way, without the monotonic time.
		buf = append(buf, a.Key...)
		buf = append(buf, "="...)
		buf = a.Value.Time().AppendFormat(buf, time.RFC3339Nano)
		buf = append(buf, ' ')
	case slog.KindGroup:
		attrs := a.Value.Group()
		// Ignore empty groups.
		if len(attrs) == 0 {
			return buf
		}
		// If the key is non-empty, write it out and indent the rest of the attrs.
		// Otherwise, inline the attrs.
		prefix := a.Key
		if a.Key != "" {
			prefix += "."
		}
		for _, ga := range attrs {
			buf = h.appendAttr(buf, slog.Attr{
				Key:   prefix + ga.Key,
				Value: ga.Value,
			})
		}
	default:
		buf = append(buf, a.Key...)
		buf = append(buf, "="...)
		buf = append(buf, a.Value.String()...)
		buf = append(buf, ' ')
	}
	return buf
}
