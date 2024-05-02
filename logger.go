package log

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"runtime"
	"sync/atomic"
	"time"

	"zestack.dev/color"
)

type leveler struct {
	l *logger
}

func (l *leveler) Level() slog.Level {
	return l.l.Level().Level()
}

type writer struct {
	l *logger
}

func (w *writer) Write(p []byte) (n int, err error) {
	return w.l.Output().Write(p)
}

func (w *writer) Fd() uintptr {
	o := w.l.Output()
	if x, ok := o.(interface{ Fd() uintptr }); ok {
		return x.Fd()
	}
	return 0
}

type logger struct {
	level   atomic.Int32 // Level
	out     atomic.Value // io.Writer
	handler atomic.Value // slog.Handler
}

func defaultNewHandler(w io.Writer, opts *slog.HandlerOptions) slog.Handler {
	writer, ok := w.(color.Writer)
	if !ok {
		writer = color.NewWriter(w)
	}
	return NewTextHandler(writer, opts)
}

func New(opts *Options) Logger {
	if opts == nil {
		opts = &Options{
			Level: LevelInfo,
		}
	}
	if opts.Writer == nil {
		opts.Writer = os.Stderr
	}
	if opts.NewHandler == nil {
		opts.NewHandler = defaultNewHandler
	}

	l := new(logger)
	l.SetLevel(opts.Level)
	l.SetOutput(opts.Writer)
	l.SetHandler(opts.NewHandler(&writer{l}, &slog.HandlerOptions{
		AddSource:   opts.AddSource,
		Level:       &leveler{l},
		ReplaceAttr: opts.ReplaceAttr,
	}))

	return l
}

func (l *logger) Output() io.Writer {
	return l.out.Load().(io.Writer)
}

func (l *logger) SetOutput(w io.Writer) {
	l.out.Store(w)
}

// Handler returns l's Handler.
func (l *logger) Handler() slog.Handler {
	return l.handler.Load().(slog.Handler)
}

func (l *logger) SetHandler(h slog.Handler) {
	l.handler.Store(h)
}

// Level 返回开启的日志等级
func (l *logger) Level() Level {
	return Level(l.level.Load())
}

// SetLevel 设置开启的日志等级
func (l *logger) SetLevel(level Level) {
	l.level.Store(int32(level))
}

// Enabled 判断指定的日志级别是否开启
func (l *logger) Enabled(ctx context.Context, level Level) bool {
	if ctx == nil {
		ctx = context.Background()
	}
	return l.Handler().Enabled(ctx, level.Level())
}

func (l *logger) clone(h slog.Handler) *logger {
	c := new(logger)
	c.SetLevel(l.Level())
	c.SetOutput(l.Output())
	c.SetHandler(h)
	return c
}

func (l *logger) With(args ...any) Logger {
	if len(args) == 0 {
		return l
	}
	return l.clone(l.Handler().WithAttrs(argsToAttrSlice(args)))
}

func (l *logger) WithGroup(name string) Logger {
	if name == "" {
		return l
	}
	return l.clone(l.Handler().WithGroup(name))
}

func (l *logger) log(ctx context.Context, level Level, msg any, args []any) string {
	if !l.Enabled(ctx, level) {
		if level != LevelPanic {
			return ""
		}
		var sprintArgs []any
		var format string
		if _, ok := msg.(Attr); ok {
			// ignore
		} else if s, ok := msg.(string); ok {
			format = s
		} else {
			sprintArgs = append(sprintArgs, msg)
		}
		if len(args) > 0 {
			for _, arg := range args {
				switch arg.(type) {
				case Attr:
					// ignore
				default:
					sprintArgs = append(sprintArgs, arg)
				}
			}
		}
		if format == "" {
			return fmt.Sprint(sprintArgs...)
		}
		return fmt.Sprintf(format, sprintArgs...)
	}

	var pc uintptr
	var pcs [1]uintptr
	// skip [runtime.Callers, this function, this function's caller]
	runtime.Callers(3, pcs[:])
	pc = pcs[0]

	r := slog.NewRecord(time.Now(), level.Level(), "", pc)

	var sprintArgs []any
	var attrs []Attr
	var format string

	if a, ok := msg.(Attr); ok {
		attrs = append(attrs, a)
	} else if s, ok := msg.(string); ok {
		format = s
	} else {
		sprintArgs = append(sprintArgs, msg)
	}

	if len(args) > 0 {
		for _, arg := range args {
			switch v := arg.(type) {
			case Attr:
				attrs = append(attrs, v)
			default:
				sprintArgs = append(sprintArgs, arg)
			}
		}
	}

	if format == "" {
		r.Message = fmt.Sprint(sprintArgs...)
	} else {
		r.Message = fmt.Sprintf(format, sprintArgs...)
	}
	str := r.Message

	if len(attrs) > 0 {
		r.AddAttrs(attrs...)
	}

	if ctx == nil {
		ctx = context.Background()
	}

	_ = l.Handler().Handle(ctx, r)

	return str
}

func (l *logger) Log(level Level, msg any, args ...any) {
	l.log(nil, level, msg, args)
}

func (l *logger) Trace(msg any, args ...any) {
	l.log(nil, LevelTrace, msg, args)
}

func (l *logger) Debug(msg any, args ...any) {
	l.log(nil, LevelDebug, msg, args)
}

func (l *logger) Info(msg any, args ...any) {
	l.log(nil, LevelInfo, msg, args)
}

func (l *logger) Warn(msg any, args ...any) {
	l.log(nil, LevelWarn, msg, args)
}

func (l *logger) Error(msg any, args ...any) {
	l.log(nil, LevelError, msg, args)
}

func (l *logger) Panic(msg any, args ...any) {
	panic(l.log(nil, LevelPanic, msg, args))
}

func (l *logger) Fatal(msg any, args ...any) {
	l.log(nil, LevelFatal, msg, args)
	os.Exit(1)
}
