package builtin

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"github.com/WJQSERVER-STUDIO/go-utils/iox"
	"os"
	"path/filepath"
	"strings"
)

func init() {
	RegisterBuiltin(&BuiltinCommand{
		Name:        "mv",
		Description: "移动或重命名文件或目录",
		Usage:       "mv [-f] [-i] [-n] [-v] source... destination",
		Help: `移动或重命名文件/目录；跨文件系统失败时回退到复制再删除。

选项:
  -f, --force           覆盖前不提示
  -i, --interactive     覆盖前提示确认
  -n, --no-clobber      不覆盖已存在的文件
  -v, --verbose         显示移动过程

示例:
  mv file.txt dest/
  mv old.txt new.txt
  mv -n file.txt dest/    # 不覆盖
  mv -v file.txt dest/    # 显示移动过程`,
		Action: Mv,
	})
}

type mvOptions struct {
	force       bool
	interactive bool
	noClobber   bool
	verbose     bool
}

func Mv(args []string, env Environment, stdin io.Reader, stdout io.Writer, stderr io.Writer) int {
	args = PreprocessArgs(args)
	fs := flag.NewFlagSet("mv", flag.ContinueOnError)
	fs.SetOutput(stderr)

	opts := &mvOptions{}
	fs.BoolVar(&opts.force, "f", false, "do not prompt before overwriting")
	fs.BoolVar(&opts.force, "force", false, "do not prompt before overwriting")
	fs.BoolVar(&opts.interactive, "i", false, "prompt before overwrite")
	fs.BoolVar(&opts.interactive, "interactive", false, "prompt before overwrite")
	fs.BoolVar(&opts.noClobber, "n", false, "do not overwrite an existing file")
	fs.BoolVar(&opts.noClobber, "no-clobber", false, "do not overwrite an existing file")
	fs.BoolVar(&opts.verbose, "v", false, "explain what is being done")
	fs.BoolVar(&opts.verbose, "verbose", false, "explain what is being done")

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

	destInfo, destErr := os.Lstat(dest)
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

		// 检查目标文件是否存在
		if _, err := os.Lstat(actualDest); err == nil {
			// 目标文件存在
			if opts.noClobber {
				// -n: 不覆盖
				if opts.verbose {
					fmt.Fprintf(stdout, "mv: not moving '%s' to '%s' (no-clobber)\n", src, actualDest)
				}
				continue
			}

			if opts.interactive && !opts.force {
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

		// 执行移动
		if opts.verbose {
			fmt.Fprintf(stdout, "mv: moving '%s' -> '%s'\n", src, actualDest)
		}

		err := os.Rename(src, actualDest)
		if err != nil {
			// 跨文件系统失败时回退到复制再删除
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

	_, err = iox.Copy(d, s)
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
