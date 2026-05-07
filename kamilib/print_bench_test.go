package kamilib

import (
	"fmt"
	"io"
	"strconv"
	"testing"

	"github.com/valyala/bytebufferpool"
)

var sink int

// ---------- fmt 基准（旧方式） ----------

func BenchmarkFmtPrintln_String(b *testing.B) {
	for b.Loop() {
		fmt.Fprintln(io.Discard, "hello world")
	}
}

func BenchmarkFmtPrintln_Int(b *testing.B) {
	for b.Loop() {
		fmt.Fprintln(io.Discard, 1234567890)
	}
}

func BenchmarkFmtPrintln_IntFormat(b *testing.B) {
	for b.Loop() {
		fmt.Fprintln(io.Discard, strconv.FormatInt(1234567890, 10))
	}
}

func BenchmarkFmtPrintln_Bool(b *testing.B) {
	for b.Loop() {
		fmt.Fprintln(io.Discard, true)
	}
}

// ---------- kamilib 基准（新方式） ----------

func BenchmarkKamilibWritePrint_String(b *testing.B) {
	for b.Loop() {
		WritePrint(io.Discard, "hello world")
	}
}

func BenchmarkKamilibWritePrint_Int(b *testing.B) {
	for b.Loop() {
		bb := bytebufferpool.Get()
		bb.B = strconv.AppendInt(bb.B, 1234567890, 10)
		bb.B = append(bb.B, '\n')
		bb.WriteTo(io.Discard)
		bytebufferpool.Put(bb)
	}
}

func BenchmarkKamilibWritePrint_Bool(b *testing.B) {
	for b.Loop() {
		bb := bytebufferpool.Get()
		bb.B = strconv.AppendBool(bb.B, true)
		bb.B = append(bb.B, '\n')
		bb.WriteTo(io.Discard)
		bytebufferpool.Put(bb)
	}
}

func BenchmarkKamilibKamiPrint_String(b *testing.B) {
	old := Stdout
	Stdout = io.Discard
	defer func() { Stdout = old }()

	for b.Loop() {
		KamiPrint("hello world")
	}
}

func BenchmarkKamilibKamiPrint_Int(b *testing.B) {
	old := Stdout
	Stdout = io.Discard
	defer func() { Stdout = old }()

	for b.Loop() {
		KamiPrint(int64(1234567890))
	}
}

func BenchmarkKamilibAppendAny_String(b *testing.B) {
	var buf []byte
	for b.Loop() {
		buf = AppendAny(buf[:0], "hello world")
	}
	sink = len(buf)
}

func BenchmarkKamilibAppendAny_Int(b *testing.B) {
	var buf []byte
	for b.Loop() {
		buf = AppendAny(buf[:0], int64(1234567890))
	}
	sink = len(buf)
}
