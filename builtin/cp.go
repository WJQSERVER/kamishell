package builtin

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
)

func init() {
	RegisterBuiltin("cp", Cp)
}

func Cp(args []string, env Environment, stdin io.Reader, stdout io.Writer, stderr io.Writer) int {
	if len(args) < 2 {
		fmt.Fprintln(stderr, "cp: missing file operand")
		return 1
	}

	src := args[0]
	dst := args[1]

	// If dst is a directory, copy src into it
	dstInfo, err := os.Stat(dst)
	if err == nil && dstInfo.IsDir() {
		dst = filepath.Join(dst, filepath.Base(src))
	}

	err = copyFile(src, dst)
	if err != nil {
		fmt.Fprintf(stderr, "cp: %v\n", err)
		return 1
	}

	return 0
}

func copyFile(src, dst string) error {
	s, err := os.Open(src)
	if err != nil {
		return err
	}
	defer s.Close()

	info, err := s.Stat()
	if err != nil {
		return err
	}
	if info.IsDir() {
		return fmt.Errorf("%s is a directory (recursive copy not supported)", src)
	}

	d, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer d.Close()

	_, err = io.Copy(d, s)
	if err != nil {
		return err
	}

	return os.Chmod(dst, info.Mode())
}
