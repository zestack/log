package log

import (
	"log/slog"
	"sync"
	"zestack.dev/color"
)

var (
	cError = color.New(color.FgHiRed, color.Bold)
	cInfo  = color.New(color.FgHiGreen, color.Bold)
	cWarn  = color.New(color.FgHiYellow, color.Bold)
	cFatal = color.New(color.FgHiBlue, color.Bold)
	cPanic = color.New(color.FgHiMagenta, color.Bold)
	cDebug = color.New(color.FgHiCyan, color.Bold)
	cTrace = color.New(color.FgHiCyan, color.Bold)
)

func levelToString(l slog.Level) string {
	return parseSlogLevel(l).String()
}

func levelToColor(l slog.Level) (*color.Value, string) {
	switch level := parseSlogLevel(l); level {
	case LevelTrace:
		return cTrace.Wrap(level), ""
	case LevelDebug:
		return cDebug.Wrap(level), ""
	case LevelInfo:
		return cInfo.Wrap(level.String()), " "
	case LevelWarn:
		return cWarn.Wrap(level.String()), " "
	case LevelError:
		return cError.Wrap(level.String()), ""
	case LevelPanic:
		return cPanic.Wrap(level.String()), ""
	case LevelFatal:
		return cFatal.Wrap(level.String()), ""
	default:
		if level < LevelTrace {
			return cTrace.Wrap(level.String()), ""
		} else {
			return cPanic.Wrap(level.String()), ""
		}
	}
}

var bufPool = sync.Pool{
	New: func() any {
		b := make([]byte, 0, 1024)
		return &b
	},
}

func allocBuf() *[]byte {
	return bufPool.Get().(*[]byte)
}

func freeBuf(b *[]byte) {
	// To reduce peak allocation, return only smaller buffers to the pool.
	const maxBufferSize = 16 << 10
	if cap(*b) <= maxBufferSize {
		*b = (*b)[:0]
		bufPool.Put(b)
	}
}
