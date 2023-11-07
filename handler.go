package log

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync"
	"sync/atomic"
)

type color int
type scheme [2]int

const escape = "\x1b"

func (u color) wrap(s string) string {
	return fmt.Sprintf("%s[%dm%s%s[0m", escape, u, s, escape)
}

func (u scheme) wrap(s string) string {
	return fmt.Sprintf("%s[%d;%dm%s%s[0m", escape, u[0], u[1], s, escape)
}

var cls = [...]scheme{
	{107, 30}, // 白底黑字，T -> Trace
	{104, 97}, // 蓝底白字，D -> Debug
	{102, 97}, // 绿底白字，I -> Info
	{103, 30}, // 黄低黑字，W -> Warn
	{101, 97}, // 红底黑字，E -> Error
	{105, 97}, // 洋低白字，P -> Panic
	{106, 97}, // 青底白字，F -> Fatal
	//{100, 97}, // 黑底白字
}

func colorLevel(colorful bool, l Level) string {
	if !colorful {
		return l.String()
	}
	return cls[l].wrap(" " + l.String()[:1] + " ")
}

func colorContent(colorful bool, l Level, s string) string {
	if !colorful || l < LevelWarn {
		return s
	}
	return color(cls[l][0] - 10).wrap(s)
}

// 全局缓存前缀
// TODO(hupeh): 使用 WeakMap
var prefixToColor sync.Map
var prefixLength int32

func colorPrefix(colorful bool, s string) string {
	count := int(atomic.LoadInt32(&prefixLength))
	if l := len(s); count < l {
		if l < 9 {
			l = 9
		}
		atomic.StoreInt32(&prefixLength, int32(l+1))
		count = l + 1
	}
	s = overflow(s, count)
	if !colorful {
		return s
	}
	if v, ok := prefixToColor.Load(s); ok {
		return v.(color).wrap(s)
	}
	x := int32(0)
	for _, c := range s {
		x = ((x << 5) - x) + c
		x |= 0 // Convert to 32bit integer
	}
	i := x % 12
	if i < 0 {
		i = -i
	}
	if i < 6 {
		i += 31
	} else {
		i += -6 + 91
	}
	v := color(i)
	prefixToColor.Store(s, v)
	return v.wrap(s)
}

func colorSuffix(colorful bool, s string) string {
	if colorful {
		return color(2).wrap(s) // dim
	}
	return s
}

func noColorIsSet() bool {
	return os.Getenv("NO_COLOR") != "" || os.Getenv("TERM") == "dumb"
}

// 该函数用于 prefix 和 funcName
// 所以不考虑中文、表情等特殊字符的子宽
func overflow(s string, n int) string {
	l := len(s)
	if l == n {
		return s
	}
	if l < n {
		for l < n {
			s += " "
			l++
		}
		return s
	}
	// 假设我们的前缀使用命名空间风格，如：`app:db`，
	// 所以这里通过 `:.` 来尝试保留完整的函数名或命名空间。
	j := strings.LastIndexAny(s, ".:")
	if j <= 0 || l-j >= n {
		j = l - 1
	}
	i := 0
	for i < j {
		x := i + 3 + (l - j)
		if x == n {
			break
		} else if x < n {
			i++
			if l-j < i {
				j--
			}
		} else {
			i -= x - n
			for i < 0 {
				j++
			}
		}
	}
	return s[:i] + "..." + s[j:]
}

//// 获取正在运行的函数名
//func funcName() string {
//	pc := make([]uintptr, 1)
//	runtime.Callers(5, pc)
//	f := runtime.FuncForPC(pc[0])
//	return f.Name()
//}

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

	colorful := w.isColorful && !noColorIsSet()

	var suffix string
	if len(attrs) > 0 {
		bts, err := json.Marshal(attrs)
		if err == nil && len(bts) > 0 {
			suffix = " " + colorSuffix(colorful, string(bts))
		}
	}

	prefix := r.Time.Format("2006-01-02 15:04:05.000 ") // 时间
	prefix += colorPrefix(colorful, r.Prefix)           // 日志前缀
	//prefix += overflow(funcName(), 30) + "  "           // 函数名称
	prefix += colorLevel(colorful, r.Level) + " " // 日志级别

	// TODO(hupeh): 如何处理换行行为？？
	//for i, line := range strings.Split(r.Message, "\n") {
	//	if i == 0 {
	//		w.Write([]byte(prefix + colorContent(colorful, r.Level, line) + suffix + "\n"))
	//	} else {
	//		n := 57
	//		if colorful {
	//			n += 3
	//		} else {
	//			n += len(r.Level.String())
	//		}
	//		w.Write([]byte(strings.Repeat(" ", n) + line + suffix + "\n"))
	//	}
	//}
	msg := strings.ReplaceAll(r.Message, "\n", "")
	msg = colorContent(colorful, r.Level, msg)
	_, err := w.Write([]byte(prefix + msg + suffix + "\n"))
	if err != nil {
		// FIXME(hupeh): 是否应该忽略这个错误？
		panic(err)
	}
}
