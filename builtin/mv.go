package builtin

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

func init() {
	RegisterBuiltin(&BuiltinCommand{
		Name:        "mv",
		Description: "移动或重命名文件或目录",
		Usage:       "mv [-f] [-i] source... destination",
		Help:        "移动或重命名文件/目录；跨文件系统失败时回退到复制再删除。",
		Action:      Mv,
	})
}

func Mv(args []string, env Environment, stdin io.Reader, stdout io.Writer, stderr io.Writer) int {
	args = PreprocessArgs(args)
	fs := flag.NewFlagSet("mv", flag.ContinueOnError)
	fs.SetOutput(stderr)

	force := fs.Bool("f", false, "do not prompt before overwriting")
	interactive := fs.Bool("i", false, "prompt before overwrite")

	if err := fs.Parse(args); err != nil {
		return 1
	}

	targets := fs.Args()
	if len(targets) < 2 {
		fmt.Fprintln(stderr, "mv: missing file operand")
		return 1
	}

	dest := targets[len(targets)-1]
	sources := targets[:len(targets)-1]

	destInfo, destErr := os.Stat(dest)
	isDestDir := destErr == nil && destInfo.IsDir()

	if len(sources) > 1 && !isDestDir {
		fmt.Fprintf(stderr, "mv: target '%s' is not a directory\n", dest)
		return 1
	}

	exitCode := 0
	reader := bufio.NewReader(stdin)

	for _, src := range sources {
		actualDest := dest
		if isDestDir {
			actualDest = filepath.Join(dest, filepath.Base(src))
		}

		if _, err := os.Stat(actualDest); err == nil && !*force {
			if *interactive {
				fmt.Fprintf(stdout, "mv: overwrite '%s'? ", actualDest)
				resp, readErr := reader.ReadString('\n')
				if readErr != nil {
					fmt.Fprintf(stderr, "mv: failed to read confirmation: %v\n", readErr)
					exitCode = 1
					continue
				}
				resp = strings.ToLower(strings.TrimSpace(resp))
				if resp != "y" && resp != "yes" {
					continue
				}
			}
		}

		err := os.Rename(src, actualDest)
		if err != nil {
			// Try copy and delete if Rename fails (e.g. across filesystems)
			err = moveByCopy(src, actualDest)
			if err != nil {
				fmt.Fprintf(stderr, "mv: %v\n", err)
				exitCode = 1
			}
		}
	}

	return exitCode
}

func moveByCopy(src, dst string) error {
	srcInfo, err := os.Lstat(src)
	if err != nil {
		return err
	}

	if srcInfo.IsDir() {
		// Use Cp implementation logic if possible, but here we just do a simplified version
		err = copyDirInternal(src, dst)
		if err != nil {
			return err
		}
		return os.RemoveAll(src)
	}

	err = copyFileSimple(src, dst)
	if err != nil {
		return err
	}
	return os.Remove(src)
}

func copyFileSimple(src, dst string) error {
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
	return err
}

func copyDirInternal(src, dst string) error {
	srcInfo, err := os.Lstat(src)
	if err != nil {
		return err
	}

	err = os.MkdirAll(dst, srcInfo.Mode())
	if err != nil {
		return err
	}

	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		err = moveByCopy(filepath.Join(src, entry.Name()), filepath.Join(dst, entry.Name()))
		if err != nil {
			return err
		}
	}
	return nil
}
