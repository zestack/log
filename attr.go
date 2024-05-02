package log

import (
	"log/slog"
	"time"
)

// Attr 引用 log.Attr，方便日后定制
type Attr = slog.Attr

// String returns an Attr for a string value.
func String(key, value string) Attr {
	return slog.String(key, value)
}

// Int64 returns an Attr for an int64.
func Int64(key string, value int64) Attr {
	return slog.Int64(key, value)
}

// Int converts an int to an int64 and returns
// an Attr with that value.
func Int(key string, value int) Attr {
	return slog.Int(key, value)
}

// Uint64 returns an Attr for an uint64.
func Uint64(key string, v uint64) Attr {
	return slog.Uint64(key, v)
}

// Float64 returns an Attr for a floating-point number.
func Float64(key string, v float64) Attr {
	return slog.Float64(key, v)
}

// Bool returns an Attr for a bool.
func Bool(key string, v bool) Attr {
	return slog.Bool(key, v)
}

// Time returns an Attr for a time.Time.
// It discards the monotonic portion.
func Time(key string, v time.Time) Attr {
	return slog.Time(key, v)
}

// Duration returns an Attr for a time.Duration.
func Duration(key string, v time.Duration) Attr {
	return slog.Duration(key, v)
}

// Group returns an Attr for a Group Instance.
// The first argument is the key; the remaining arguments
// are converted to Attrs as in [Logger.Log].
//
// Use Group to collect several key-value pairs under a single
// key on a log line, or as the result of LogValue
// in order to log a single value as multiple Attrs.
func Group(key string, args ...any) Attr {
	return slog.Group(key, args...)
}

// Any returns an Attr for the supplied value.
// See [AnyValue] for how values are treated.
func Any(key string, value any) Attr {
	return slog.Any(key, value)
}

const badKey = "!BADKEY"

// argsToAttr turns a prefix of the nonempty args slice into an Attr
// and returns the unconsumed portion of the slice.
// If args[0] is an Attr, it returns it.
// If args[0] is a string, it treats the first two elements as
// a key-value pair.
// Otherwise, it treats args[0] as a value with a missing key.
func argsToAttr(args []any) (Attr, []any) {
	switch x := args[0].(type) {
	case string:
		if len(args) == 1 {
			return String(badKey, x), nil
		}
		return Any(x, args[1]), args[2:]

	case Attr:
		return x, args[1:]

	default:
		return Any(badKey, x), args[1:]
	}
}

func argsToAttrSlice(args []any) []Attr {
	var (
		attr  Attr
		attrs []Attr
	)
	for len(args) > 0 {
		attr, args = argsToAttr(args)
		attrs = append(attrs, attr)
	}
	return attrs
}
