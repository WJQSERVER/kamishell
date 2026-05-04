package kamilib

import (
	"fmt"
	"io"
	"os"
	"strconv"

	"github.com/valyala/bytebufferpool"
)

// AppendAny 将任意值以文本形式追加到 b，已知类型走 strconv 避免反射。
// 未知类型回退到 fmt.Sprint。
func AppendAny(b []byte, v any) []byte {
	switch x := v.(type) {
	case string:
		return append(b, x...)
	case []byte:
		return append(b, x...)
	case int64:
		return strconv.AppendInt(b, x, 10)
	case int:
		return strconv.AppendInt(b, int64(x), 10)
	case float64:
		return strconv.AppendFloat(b, x, 'f', -1, 64)
	case bool:
		return strconv.AppendBool(b, x)
	case nil:
		return append(b, "nil"...)
	case fmt.Stringer:
		return append(b, x.String()...)
	default:
		return append(b, fmt.Sprint(x)...)
	}
}

// Stdout 是编译产物的默认输出目标，方便测试时替换。
var Stdout io.Writer = os.Stdout

// KamiPrint 将 v 的文本表示写入 Stdout 并追加换行。
// 用于编译产物中 print 关键字的运行时实现。
func KamiPrint(v any) error {
	bb := bytebufferpool.Get()
	bb.B = AppendAny(bb.B, v)
	bb.B = append(bb.B, '\n')
	_, err := bb.WriteTo(Stdout)
	bytebufferpool.Put(bb)
	return err
}

// WritePrint 将 s 写入 w 并追加换行，使用 buf 池避免分配。
func WritePrint(w io.Writer, s string) (int, error) {
	bb := bytebufferpool.Get()
	bb.B = append(bb.B, s...)
	bb.B = append(bb.B, '\n')
	n, err := w.Write(bb.B)
	bytebufferpool.Put(bb)
	return n, err
}

// AppendPrint 将 v 的文本表示追加到 b 并追加换行。
func AppendPrint(b []byte, v any) []byte {
	b = AppendAny(b, v)
	return append(b, '\n')
}
