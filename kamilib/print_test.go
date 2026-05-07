package kamilib

import (
	"bytes"
	"testing"
)

func TestAppendAny(t *testing.T) {
	tests := []struct {
		name string
		v    any
		want string
	}{
		{"string", "hello", "hello"},
		{"int64", int64(42), "42"},
		{"int", int(-7), "-7"},
		{"float64", float64(3.14), "3.14"},
		{"bool true", true, "true"},
		{"bool false", false, "false"},
		{"nil", nil, "nil"},
		{"bytes", []byte("abc"), "abc"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := string(AppendAny(nil, tt.v))
			if got != tt.want {
				t.Errorf("AppendAny() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestAppendPrint(t *testing.T) {
	got := string(AppendPrint(nil, int64(99)))
	want := "99\n"
	if got != want {
		t.Errorf("AppendPrint() = %q, want %q", got, want)
	}
}

func TestWritePrint(t *testing.T) {
	var buf bytes.Buffer
	n, err := WritePrint(&buf, "test")
	if err != nil {
		t.Fatalf("WritePrint() error: %v", err)
	}
	if got := buf.String(); got != "test\n" {
		t.Errorf("WritePrint() = %q, want %q", got, "test\n")
	}
	if n != 5 {
		t.Errorf("WritePrint() n = %d, want 5", n)
	}
}

func TestKamiPrint(t *testing.T) {
	var buf bytes.Buffer
	old := Stdout
	Stdout = &buf
	defer func() { Stdout = old }()

	if err := KamiPrint(int64(123)); err != nil {
		t.Fatalf("KamiPrint() error: %v", err)
	}
	if got := buf.String(); got != "123\n" {
		t.Errorf("KamiPrint() = %q, want %q", got, "123\n")
	}
}

func TestAppendAnyLargeInt(t *testing.T) {
	got := string(AppendAny(nil, int64(9999999999)))
	want := "9999999999"
	if got != want {
		t.Errorf("AppendAny(large int) = %q, want %q", got, want)
	}
}
