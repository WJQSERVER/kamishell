package builtin

import (
	"runtime"
	"testing"
)

func TestGetOS(t *testing.T) {
	os := GetOS()
	if os != runtime.GOOS {
		t.Errorf("GetOS() = %q, want %q", os, runtime.GOOS)
	}
}

func TestGetArch(t *testing.T) {
	arch := GetArch()
	if arch != runtime.GOARCH {
		t.Errorf("GetArch() = %q, want %q", arch, runtime.GOARCH)
	}
}
