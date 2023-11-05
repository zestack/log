package log

import (
	"io"
	"time"
)

var global = New("log")

func Default() Logger                { return global }
func SetDefault(l Logger)            { global = l }
func Prefix() string                 { return global.Prefix() }
func GetLevel() Level                { return global.Level() }
func SetLevel(level Level)           { global.SetLevel(level) }
func Enabled(level Level) bool       { return global.Enabled(level) }
func Timezone() *time.Location       { return global.Timezone() }
func SetTimezone(loc *time.Location) { global.SetTimezone(loc) }
func Output() io.Writer              { return global.Output() }
func SetOutput(w io.Writer)          { global.SetOutput(w) }

func With(attrs ...Attr) Logger                      { return global.With(attrs...) }
func WithPrefix(prefix string, attrs ...Attr) Logger { return global.WithPrefix(prefix, attrs...) }

func Print(i ...any)                    { global.Print(i...) }
func Printf(format string, args ...any) { global.Printf(format, args) }
func Printj(j map[string]any)           { global.Printj(j) }
func Debug(i ...any)                    { global.Debug(i...) }
func Debugf(format string, args ...any) { global.Debugf(format, args) }
func Debugj(j map[string]any)           { global.Debugj(j) }
func Info(i ...any)                     { global.Info(i...) }
func Infof(format string, args ...any)  { global.Infof(format, args) }
func Infoj(j map[string]any)            { global.Infoj(j) }
func Warn(i ...any)                     { global.Warn(i...) }
func Warnf(format string, args ...any)  { global.Warnf(format, args) }
func Warnj(j map[string]any)            { global.Warnj(j) }
func Error(i ...any)                    { global.Error(i...) }
func Errorf(format string, args ...any) { global.Errorf(format, args) }
func Errorj(j map[string]any)           { global.Errorj(j) }
func Panic(i ...any)                    { global.Panic(i...) }
func Panicf(format string, args ...any) { global.Panicf(format, args) }
func Panicj(j map[string]any)           { global.Panicj(j) }
func Fatal(i ...any)                    { global.Fatal(i...) }
func Fatalf(format string, args ...any) { global.Fatalf(format, args) }
func Fatalj(j map[string]any)           { global.Fatalj(j) }
