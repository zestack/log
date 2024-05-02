package log

import (
	"context"
	"io"
	"log/slog"
	"sync/atomic"
)

// Logger defines the logging interface.
type Logger interface {
	Output() io.Writer
	SetOutput(w io.Writer)
	Level() Level
	SetLevel(level Level)
	Enabled(ctx context.Context, level Level) bool
	// With returns a Logger that includes the given attributes
	// in each output operation. Arguments are converted to
	// attributes as if by [Logger.Log].
	With(args ...any) Logger
	// WithGroup returns a Logger that starts a group, if name is non-empty.
	// The keys of all attributes added to the Logger will be qualified by the given
	// name. (How that qualification happens depends on the [Handler.WithGroup]
	// method of the Logger's Handler.)
	//
	// If name is empty, WithGroup returns the receiver.
	WithGroup(name string) Logger
	// Log emits a log record with the current time and the given level and message.
	// The Record's Attrs consist of the Logger's attributes followed by
	// the Attrs specified by args.
	//
	// The attribute arguments are processed as follows:
	//   - If an argument is an Attr, it is used as is.
	//   - If an argument is a string and this is not the last argument,
	//     the following argument is treated as the value and the two are combined
	//     into an Attr.
	//   - Otherwise, the argument is treated as a value with key "!BADKEY".
	Log(level Level, msg any, args ...any)
	// Trace logs at [LevelTrace].
	Trace(msg any, args ...any)
	// Debug logs at [LevelDebug].
	Debug(msg any, args ...any)
	// Info logs at [LevelInfo].
	Info(msg any, args ...any)
	// Warn logs at [LevelWarn].
	Warn(msg any, args ...any)
	// Error logs at [LevelError].
	Error(msg any, args ...any)
	// Panic logs at [LevelPanic].
	Panic(msg any, args ...any)
	// Fatal logs at [LevelFatal].
	Fatal(msg any, args ...any)
}

type Options struct {
	// AddSource causes the handler to compute the source code position
	// of the log statement and add a SourceKey attribute to the output.
	AddSource bool

	// Level reports the minimum record level that will be logged.
	// The handler discards records with lower levels.
	// If Level is nil, the handler assumes LevelInfo.
	// The handler calls Level.Level for each record processed;
	// to adjust the minimum level dynamically, use a LevelVar.
	Level Level

	// ReplaceAttr is called to rewrite each non-group attribute before it is logged.
	// The attribute's value has been resolved (see [Value.Resolve]).
	// If ReplaceAttr returns a zero Attr, the attribute is discarded.
	//
	// The built-in attributes with keys "time", "level", "source", and "msg"
	// are passed to this function, except that time is omitted
	// if zero, and source is omitted if AddSource is false.
	//
	// The first argument is a list of currently open groups that contain the
	// Attr. It must not be retained or modified. ReplaceAttr is never called
	// for Group attributes, only their contents. For example, the attribute
	// list
	//
	//     Int("a", 1), Group("g", Int("b", 2)), Int("c", 3)
	//
	// results in consecutive calls to ReplaceAttr with the following arguments:
	//
	//     nil, Int("a", 1)
	//     []string{"g"}, Int("b", 2)
	//     nil, Int("c", 3)
	//
	// ReplaceAttr can be used to change the default keys of the built-in
	// attributes, convert types (for example, to replace a `time.Time` with the
	// integer seconds since the Unix epoch), sanitize personal information, or
	// remove attributes from the output.
	ReplaceAttr func(groups []string, a Attr) Attr

	// 前端日志写入接口
	Writer io.Writer

	NewHandler func(w io.Writer, opts *slog.HandlerOptions) slog.Handler
}

var defaultLogger atomic.Value

func init() {
	defaultLogger.Store(New(nil))
}

func Default() Logger {
	return defaultLogger.Load().(Logger)
}

func SetDefault(l Logger) {
	defaultLogger.Store(l)
}

func GetLevel() Level {
	return Default().Level()
}

func SetLevel(level Level) {
	Default().SetLevel(level)
}

func Enabled(ctx context.Context, level Level) bool {
	return Default().Enabled(ctx, level)
}

func Output() io.Writer {
	return Default().Output()
}

func SetOutput(w io.Writer) {
	Default().SetOutput(w)
}

func With(args ...any) Logger {
	return Default().With(args...)
}

func WithGroup(name string) Logger {
	return Default().WithGroup(name)
}

func Log(level Level, msg any, args ...any) {
	Default().Log(level, msg, args...)
}

func Trace(msg any, args ...any) { Default().Trace(msg, args...) }
func Debug(msg any, args ...any) { Default().Debug(msg, args...) }
func Info(msg any, args ...any)  { Default().Info(msg, args...) }
func Warn(msg any, args ...any)  { Default().Warn(msg, args...) }
func Error(msg any, args ...any) { Default().Error(msg, args...) }
func Panic(msg any, args ...any) { Default().Panic(msg, args...) }
func Fatal(msg any, args ...any) { Default().Fatal(msg, args...) }
