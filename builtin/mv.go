package builtin

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
)

func init() {
	RegisterBuiltin("mv", Mv)
}

func Mv(args []string, env Environment, stdin io.Reader, stdout io.Writer, stderr io.Writer) int {
	if len(args) < 2 {
		fmt.Fprintln(stderr, "mv: missing file operand")
		return 1
	}

	src := args[0]
	dst := args[1]

	// If dst is a directory, move src into it
	dstInfo, err := os.Stat(dst)
	if err == nil && dstInfo.IsDir() {
		dst = filepath.Join(dst, filepath.Base(src))
	}

	err = os.Rename(src, dst)
	if err != nil {
		// Try copy and delete if Rename fails (e.g. across filesystems)
		err = copyAndDelete(src, dst)
		if err != nil {
			fmt.Fprintf(stderr, "mv: %v\n", err)
			return 1
		}
	}

	return 0
}

func copyAndDelete(src, dst string) error {
	// Re-using copy logic if needed, but for now let's keep it simple
	// and assume it's just Rename for most cases.
	// If we wanted to be robust, we'd implement full copy+delete here.
	s, err := os.Open(src)
	if err != nil {
		return err
	}
	defer s.Close()

	d, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer d.Close()

	_, err = io.Copy(d, s)
	if err != nil {
		return err
	}

	s.Close()
	return os.Remove(src)
}
