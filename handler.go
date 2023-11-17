package log

import (
	"bytes"
	"encoding/json"
	"strings"
)

func fixedLength(s string, n int) string {
	l := len(s)
	if l == n {
		return s
	} else if l > n {
		return "..." + s[l-n+3:]
	} else {
		return strings.Repeat(" ", n-l) + s
	}
}

func colorize(w *Writer, l Level) (string, func(string, ...string) string) {
	var fn func(any, ...string) string
	var ls string
	switch l {
	case LevelDebug:
		fn = w.Blue
	case LevelInfo:
		fn = w.Green
		ls = " "
	case LevelWarn:
		fn = w.Yellow
		ls = " "
	case LevelError:
		fn = w.Red
	case LevelPanic:
		fn = w.Magenta
	case LevelFatal:
		fn = w.Cyan
	default:
		// ignore
	}
	hd := func(s string, s2 ...string) string {
		if fn == nil {
			return s
		}
		return fn(s, s2...)
	}
	ls = w.Dim("[") + hd(l.String()) + w.Dim("]") + ls + " "
	return ls, hd
}

func handle(w *Writer, r Record) {
	var attrs map[string]any
	for _, a := range r.Attrs {
		if a.Key != "" {
			if attrs == nil {
				attrs = make(map[string]any)
			}
			attrs[a.Key] = a.Value.Any()
		}
	}

	var suffix string
	if len(attrs) > 0 {
		bts, err := json.Marshal(attrs)
		if err == nil && len(bts) > 0 {
			suffix = " " + w.Dim(string(bts))
		}
	}

	level, colorer := colorize(w, r.Level)

	prefix := r.Time.Format("2006-01-02 15:04:05.000 ")
	prefix += w.White(fixedLength(r.Prefix, 10)) + " "
	prefix += level

	b := new(bytes.Buffer)
	lines := strings.Split(r.Message, "\n")

	for _, line := range lines {
		b.WriteString(prefix)
		b.WriteString(colorer(line))
		b.WriteString(suffix)
		b.WriteString("\n")
	}

	w.Write(b.Bytes())
}
