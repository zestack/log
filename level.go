package log

import (
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
)

type Level int

const (
	LevelTrace Level = iota
	LevelDebug       // 用于程序调试
	LevelInfo        // 用于程序运行
	LevelWarn        // 潜在错误或非预期结果
	LevelError       // 发生错误，但不影响系统的继续运行
	LevelPanic
	LevelFatal

	// 我们将 LevelInfo 与 log.LevelInfo 对应，
	// 那么自 LevelInfo 到 LevelFatal 的值就是 16
	lex = (LevelFatal - LevelInfo) * 4
)

// MarshalJSON 实现 [encoding/json.Marshaler] 接口，
// 使用双引号包裹 [Level.String] 的结果作为返回值。
func (l Level) MarshalJSON() ([]byte, error) {
	return strconv.AppendQuote(nil, l.String()), nil
}

// UnmarshalJSON 实现 [encoding/json.Unmarshaler] 接口
// It accepts any string produced by [Level.MarshalJSON],
// ignoring a case.
// It also accepts numeric offsets that would result in a different string on
// output. For example, "Error-8" would marshal as "INFO".
func (l *Level) UnmarshalJSON(data []byte) error {
	s, err := strconv.Unquote(string(data))
	if err != nil {
		return err
	}
	v, err := parseStringLevel(s)
	if err != nil {
		return err
	}
	*l = v
	return nil
}

// MarshalText 实现 [encoding.TextMarshaler] 接口
func (l Level) MarshalText() ([]byte, error) {
	return []byte(l.String()), nil
}

// UnmarshalText 实现 [encoding.TextUnmarshaler] 接口。
func (l *Level) UnmarshalText(data []byte) error {
	v, err := parseStringLevel(string(data))
	if err == nil {
		*l = v
	}
	return err
}

// Level 实现 [slog.Leveler] 接口
func (l Level) Level() slog.Level {
	return slog.Level(int(lex) - int(LevelFatal-l)*4)
}

// String 返回字符串形式
func (l Level) String() string {
	switch l {
	case LevelTrace:
		return "TRACE"
	case LevelDebug:
		return "DEBUG"
	case LevelInfo:
		return "INFO"
	case LevelWarn:
		return "WARN"
	case LevelError:
		return "ERROR"
	case LevelPanic:
		return "PANIC"
	case LevelFatal:
		return "FATAL"
	default:
		if l < LevelTrace {
			return fmt.Sprintf("TRACE-%d", l-LevelTrace)
		}
		return fmt.Sprintf("FATAL+%d", l-LevelFatal)
	}
}

// 将 log.Level 转换成日志级别
func parseSlogLevel(l slog.Level) Level {
	return Level(int(l/4) + int(LevelInfo))
}

// 字符串转日志级别
func parseStringLevel(s string) (l Level, err error) {
	name := s
	offset := 0
	if i := strings.IndexAny(s, "+-"); i >= 0 {
		name = s[:i]
		offset, err = strconv.Atoi(s[i:])
		if err != nil {
			return
		}
	}
	switch strings.ToUpper(name) {
	case "TRACE":
		l = LevelTrace
	case "DEBUG":
		l = LevelDebug
	case "INFO":
		l = LevelInfo
	case "WARN":
		l = LevelWarn
	case "ERROR":
		l = LevelError
	case "PANIC":
		l = LevelPanic
	case "FATAL":
		l = LevelFatal
	default:
		err = errors.New("unknown name")
	}
	if err == nil && offset != 0 {
		l += Level(offset)
	}
	return
}
