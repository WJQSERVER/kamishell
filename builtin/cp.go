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
		Name:        "cp",
		Description: "复制文件或目录",
		Usage:       "cp [-r|-R] [-p] [-f] [-i] [-n] [-u] [-v] source... destination",
		Help: `复制文件或目录；支持递归复制、保留模式和交互式覆盖确认。

选项:
  -r, -R, --recursive    递归复制目录
  -p, --preserve          保留文件属性（模式、时间戳）
  -f, --force             强制覆盖，不提示
  -i, --interactive       覆盖前提示确认
  -n, --no-clobber        不覆盖已存在的文件
  -u, --update            仅在源文件较新或目标不存在时复制
  -v, --verbose           显示复制过程

示例:
  cp file.txt dest/
  cp -r src/ dest/
  cp -n file.txt dest/     # 不覆盖
  cp -u file.txt dest/     # 仅更新较新的文件
  cp -v file.txt dest/     # 显示复制过程`,
		Action: Cp,
	})
}

type cpOptions struct {
	recursive   bool
	preserve    bool
	force       bool
	interactive bool
	noClobber   bool
	update      bool
	verbose     bool
}

func Cp(args []string, env Environment, stdin io.Reader, stdout io.Writer, stderr io.Writer) int {
	args = PreprocessArgs(args)
	fs := flag.NewFlagSet("cp", flag.ContinueOnError)
	fs.SetOutput(stderr)

	opts := &cpOptions{}
	fs.BoolVar(&opts.recursive, "r", false, "copy directories recursively")
	fs.BoolVar(&opts.recursive, "R", false, "copy directories recursively")
	fs.BoolVar(&opts.recursive, "recursive", false, "copy directories recursively")
	fs.BoolVar(&opts.preserve, "p", false, "preserve file attributes")
	fs.BoolVar(&opts.preserve, "preserve", false, "preserve file attributes")
	fs.BoolVar(&opts.force, "f", false, "if an existing destination file cannot be opened, remove it and try again")
	fs.BoolVar(&opts.force, "force", false, "if an existing destination file cannot be opened, remove it and try again")
	fs.BoolVar(&opts.interactive, "i", false, "prompt before overwrite")
	fs.BoolVar(&opts.interactive, "interactive", false, "prompt before overwrite")
	fs.BoolVar(&opts.noClobber, "n", false, "do not overwrite an existing file")
	fs.BoolVar(&opts.noClobber, "no-clobber", false, "do not overwrite an existing file")
	fs.BoolVar(&opts.update, "u", false, "copy only when the SOURCE file is newer than the destination file or when the destination file is missing")
	fs.BoolVar(&opts.update, "update", false, "copy only when the SOURCE file is newer than the destination file or when the destination file is missing")
	fs.BoolVar(&opts.verbose, "v", false, "explain what is being done")
	fs.BoolVar(&opts.verbose, "verbose", false, "explain what is being done")

	if err := fs.Parse(args); err != nil {
		return 1
	}

	// 处理选项互斥关系：-i 和 -n 互斥，-i 和 -f 互斥
	// 如果同时指定，最后一个生效
	// 这里简化处理：-n 优先级最高，其次是 -i，然后是 -f

	targets := fs.Args()
	if len(targets) < 2 {
		fmt.Fprintln(stderr, "cp: missing file operand")
		return 1
	}

	dest := targets[len(targets)-1]
	sources := targets[:len(targets)-1]

	destInfo, destErr := os.Lstat(dest)
	isDestDir := destErr == nil && destInfo.IsDir()

	if len(sources) > 1 && !isDestDir {
		fmt.Fprintf(stderr, "cp: target '%s' is not a directory\n", dest)
		return 1
	}

	exitCode := 0
	reader := bufio.NewReader(stdin)

	for _, src := range sources {
		actualDest := dest
		if isDestDir {
			actualDest = filepath.Join(dest, filepath.Base(src))
		}

		err := doCopy(src, actualDest, opts, reader, stdout, stderr)
		if err != nil {
			fmt.Fprintf(stderr, "cp: %v\n", err)
			exitCode = 1
		}
	}

	return exitCode
}

func doCopy(src, dst string, opts *cpOptions, reader *bufio.Reader, stdout, stderr io.Writer) error {
	srcInfo, err := os.Lstat(src)
	if err != nil {
		return err
	}

	if srcInfo.IsDir() {
		if !opts.recursive {
			return fmt.Errorf("-r not specified; omitting directory '%s'", src)
		}
		return copyDir(src, dst, opts, reader, stdout, stderr)
	}

	// File copy - check destination
	dstInfo, err := os.Lstat(dst)
	if err == nil {
		// 目标文件存在
		if opts.noClobber {
			// -n: 不覆盖
			if opts.verbose {
				fmt.Fprintf(stdout, "cp: not overwriting '%s' (no-clobber)\n", dst)
			}
			return nil
		}

		if opts.update {
			// -u: 仅在源文件较新时复制
			if !srcInfo.ModTime().After(dstInfo.ModTime()) {
				if opts.verbose {
					fmt.Fprintf(stdout, "cp: not overwriting '%s' (not newer)\n", dst)
				}
				return nil
			}
		}

		if opts.interactive {
			fmt.Fprintf(stdout, "cp: overwrite '%s'? ", dst)
			resp, readErr := reader.ReadString('\n')
			if readErr != nil {
				return fmt.Errorf("cp: failed to read confirmation: %w", readErr)
			}
			resp = strings.ToLower(strings.TrimSpace(resp))
			if resp != "y" && resp != "yes" {
				return nil
			}
		}

		if opts.force {
			os.Remove(dst)
		}
	}

	// 执行复制
	if opts.verbose {
		fmt.Fprintf(stdout, "cp: copying '%s' -> '%s'\n", src, dst)
	}

	return copyFileInternal(src, dst, srcInfo, opts.preserve)
}

func copyFileInternal(src, dst string, srcInfo os.FileInfo, preserve bool) error {
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

	if preserve {
		if err := os.Chmod(dst, srcInfo.Mode()); err != nil {
			return err
		}
		atime, mtime := currentFileTimes(srcInfo)
		if err := os.Chtimes(dst, atime, mtime); err != nil {
			return err
		}
	}
	return nil
}

func copyDir(src, dst string, opts *cpOptions, reader *bufio.Reader, stdout, stderr io.Writer) error {
	srcInfo, err := os.Lstat(src)
	if err != nil {
		return err
	}

	// 检查目标目录是否已存在
	if _, err := os.Stat(dst); err != nil {
		// 目标不存在，创建
		err = os.MkdirAll(dst, srcInfo.Mode())
		if err != nil {
			return err
		}
		if opts.verbose {
			fmt.Fprintf(stdout, "cp: created directory '%s'\n", dst)
		}
	}

	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		err = doCopy(filepath.Join(src, entry.Name()), filepath.Join(dst, entry.Name()), opts, reader, stdout, stderr)
		if err != nil {
			return err
		}
	}

	if opts.preserve {
		if err := os.Chmod(dst, srcInfo.Mode()); err != nil {
			return err
		}
		atime, mtime := currentFileTimes(srcInfo)
		if err := os.Chtimes(dst, atime, mtime); err != nil {
			return err
		}
	}
	return nil
}
