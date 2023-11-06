package log

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"os"
	"sync"
	"sync/atomic"
	"time"
	"unsafe"

	"github.com/mattn/go-isatty"
)

// Logger defines the logging interface.
type Logger interface {
	Output() io.Writer
	SetOutput(w io.Writer)
	Timezone() *time.Location
	SetTimezone(loc *time.Location)
	Prefix() string
	SetPrefix(prefix string)
	Level() Level
	SetLevel(v Level)
	Enabled(level Level) bool
	With(attrs ...Attr) Logger
	WithPrefix(prefix string, attrs ...Attr) Logger
	Print(i ...any)
	Printf(format string, args ...any)
	Printj(j map[string]any)
	Debug(i ...any)
	Debugf(format string, args ...any)
	Debugj(j map[string]any)
	Info(i ...any)
	Infof(format string, args ...any)
	Infoj(j map[string]any)
	Warn(i ...any)
	Warnf(format string, args ...any)
	Warnj(j map[string]any)
	Error(i ...any)
	Errorf(format string, args ...any)
	Errorj(j map[string]any)
	Panic(i ...any)
	Panicf(format string, args ...any)
	Panicj(j map[string]any)
	Fatal(i ...any)
	Fatalf(format string, args ...any)
	Fatalj(j map[string]any)
}

// Record 日志打印记录
type Record struct {
	Level   Level
	Prefix  string
	Time    time.Time
	Message string
	Attrs   []Attr
}

type Writer struct {
	io.Writer
	isDiscard  bool
	isColorful bool
}

func (w *Writer) IsColorful() bool {
	return w.isColorful
}

func (w *Writer) IsDiscard() bool {
	return w.isDiscard
}

// Handler 自定义日志处理器
type Handler func(w *Writer, r Record)

type logger struct {
	mu       *sync.RWMutex
	prefix   atomic.Pointer[string]
	level    atomic.Int32
	timezone unsafe.Pointer
	attrs    []Attr
	writer   *Writer
	handler  Handler
}

type Options struct {
	Level    Level
	Attrs    []Attr
	Timezone *time.Location
	Writer   io.Writer // 默认值为 os.Stderr
	Handler  Handler   // 默认值为 CommonHandler
}

func New(prefix string, opts ...Options) Logger {
	var o Options
	if len(opts) > 0 {
		o = opts[0]
	}
	if o.Writer == nil {
		o.Writer = os.Stderr
	}
	if o.Timezone == nil {
		o.Timezone = time.Local
	}
	l := &logger{
		mu:    new(sync.RWMutex),
		attrs: o.Attrs,
	}
	l.SetPrefix(prefix)
	l.SetLevel(o.Level)
	l.SetTimezone(o.Timezone)
	l.SetOutput(o.Writer)
	l.SetHandler(o.Handler)
	return l
}

func (l *logger) Output() io.Writer {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.writer
}

func (l *logger) SetOutput(w io.Writer) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.writer = &Writer{
		Writer:     w,
		isDiscard:  w == io.Discard,
		isColorful: isTerminal(w),
	}
}

func (l *logger) Timezone() *time.Location {
	// 参考 atomic.Pointer#Load 实现
	return (*time.Location)(atomic.LoadPointer(&l.timezone))
}

func (l *logger) SetTimezone(loc *time.Location) {
	// 参考 atomic.Pointer#Store 实现
	atomic.StorePointer(&l.timezone, unsafe.Pointer(loc))
}

func (l *logger) Handler() Handler {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.handler
}

func (l *logger) SetHandler(h Handler) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if h != nil {
		l.handler = h
	} else {
		l.handler = handle
	}
}

// Prefix 返回日志前缀
func (l *logger) Prefix() string {
	if p := l.prefix.Load(); p != nil {
		return *p
	}
	return ""
}

func (l *logger) SetPrefix(prefix string) {
	l.prefix.Store(&prefix)
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
func (l *logger) Enabled(level Level) bool {
	if level >= LevelPanic {
		return true
	}
	if level >= LevelTrace {
		return l.Level() <= level
	}
	return false
}

func (l *logger) With(attrs ...Attr) Logger {
	return l.WithPrefix("", attrs...)
}

func (l *logger) WithPrefix(prefix string, attrs ...Attr) Logger {
	if prefix == "" {
		if len(attrs) == 0 {
			return l
		}
		prefix = l.Prefix()
	}
	l.mu.RLock()
	if len(l.attrs) > 0 {
		attrs = append(l.attrs, attrs...)
	}
	l.mu.RUnlock()
	return New(prefix, Options{
		Level:    l.Level(),
		Attrs:    attrs,
		Timezone: l.Timezone(),
		Writer:   l.Output().(*Writer).Writer,
		Handler:  l.Handler(),
	})
}

func (l *logger) Print(i ...any) {
	l.log(LevelTrace, "", i...)
}

func (l *logger) Printf(format string, args ...any) {
	l.log(LevelTrace, format, args...)
}

func (l *logger) Printj(j map[string]any) {
	l.log(LevelTrace, "j", j)
}

func (l *logger) Debug(i ...any) {
	l.log(LevelDebug, "", i...)
}

func (l *logger) Debugf(format string, args ...any) {
	l.log(LevelDebug, format, args...)
}

func (l *logger) Debugj(j map[string]any) {
	l.log(LevelDebug, "j", j)
}

func (l *logger) Info(i ...any) {
	l.log(LevelInfo, "", i...)
}

func (l *logger) Infof(format string, args ...any) {
	l.log(LevelInfo, format, args...)
}

func (l *logger) Infoj(j map[string]any) {
	l.log(LevelInfo, "j", j)
}

func (l *logger) Warn(i ...any) {
	l.log(LevelWarn, "", i...)
}

func (l *logger) Warnf(format string, args ...any) {
	l.log(LevelWarn, format, args...)
}

func (l *logger) Warnj(j map[string]any) {
	l.log(LevelWarn, "j", j)
}

func (l *logger) Error(i ...any) {
	l.log(LevelError, "", i...)
}

func (l *logger) Errorf(format string, args ...any) {
	l.log(LevelError, format, args...)
}

func (l *logger) Errorj(j map[string]any) {
	l.log(LevelError, "j", j)
}

func (l *logger) Panic(i ...any) {
	panic(l.log(LevelPanic, "", i...))
}

func (l *logger) Panicf(format string, args ...any) {
	panic(l.log(LevelPanic, format, args...))
}

func (l *logger) Panicj(j map[string]any) {
	panic(l.log(LevelPanic, "j", j))
}

func (l *logger) Fatal(i ...any) {
	l.log(LevelFatal, "", i...)
	os.Exit(1)
}

func (l *logger) Fatalf(format string, args ...any) {
	l.log(LevelFatal, format, args...)
	os.Exit(1)
}

func (l *logger) Fatalj(j map[string]any) {
	l.log(LevelFatal, "j", j)
	os.Exit(1)
}

func (l *logger) attributes() []Attr {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.attrs[:]
}

func (l *logger) log(level Level, format string, args ...any) string {
	if !l.Enabled(level) {
		return ""
	}

	r := Record{
		Level:   level,
		Prefix:  l.Prefix(),
		Time:    time.Now().In(l.Timezone()),
		Message: format,
		Attrs:   l.attributes(),
	}
	l.mu.RLock()
	r.Attrs = l.attrs[:]
	h := l.handler
	w := l.writer
	l.mu.RUnlock()

	var sprintArgs []any
	if len(args) > 0 && format != "j" {
		for _, arg := range args {
			switch v := arg.(type) {
			case slog.Attr:
				r.Attrs = append(r.Attrs, v)
			default:
				sprintArgs = append(sprintArgs, arg)
			}
		}
	}

	switch format {
	case "":
		r.Message = fmt.Sprint(sprintArgs...)
	case "j":
		bts, err := json.Marshal(args[0])
		if err != nil {
			panic(err)
		}
		r.Message = string(bts)
	default:
		r.Message = fmt.Sprintf(r.Message, sprintArgs...)
	}

	l.mu.RLock()
	defer l.mu.RUnlock()
	h(w, r)

	return r.Message
}

func isTerminal(a any) bool {
	if f, ok := a.(interface{ Fd() uintptr }); ok {
		return isatty.IsTerminal(f.Fd()) || isatty.IsCygwinTerminal(f.Fd())
	}
	return false
}
