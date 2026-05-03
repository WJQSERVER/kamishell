//go:build linux

package builtin

import (
	"fmt"
	"os"

	"golang.org/x/sys/unix"
)

func tryCopyOnWrite(src, dst string) (bool, error) {
	s, err := os.Open(src)
	if err != nil {
		return false, fmt.Errorf("open source: %w", err)
	}
	defer s.Close()

	d, err := os.Create(dst)
	if err != nil {
		return false, fmt.Errorf("create destination: %w", err)
	}
	defer d.Close()

	err = unix.IoctlFileClone(int(d.Fd()), int(s.Fd()))
	if err != nil {
		os.Remove(dst)
		return false, fmt.Errorf("FICLONE: %w", err)
	}

	return true, nil
}
