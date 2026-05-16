//go:build freebsd

package builtin

import (
	"fmt"
	"os"
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

	srcInfo, err := s.Stat()
	if err != nil {
		os.Remove(dst)
		return false, fmt.Errorf("stat source: %w", err)
	}

	// Use copy_file_range syscall for zero-copy within same filesystem
	// On FreeBSD, this can do COW when supported by the filesystem (ZFS, UFS)
	n, err := unix.CopyFileRange(int(s.Fd()), nil, int(d.Fd()), nil, int(srcInfo.Size()), 0)
	if err != nil {
		os.Remove(dst)
		return false, fmt.Errorf("copy_file_range: %w", err)
	}
	if n != int(srcInfo.Size()) {
		os.Remove(dst)
		return false, fmt.Errorf("copy_file_range: short copy %d/%d", n, srcInfo.Size())
	}

	return true, nil
}
