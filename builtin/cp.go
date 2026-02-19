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
	RegisterBuiltin("cp", Cp)
}

func Cp(args []string, env Environment, stdin io.Reader, stdout io.Writer, stderr io.Writer) int {
	args = PreprocessArgs(args)
	fs := flag.NewFlagSet("cp", flag.ContinueOnError)
	fs.SetOutput(stderr)

	recursive := fs.Bool("r", false, "copy directories recursively")
	recursiveUpper := fs.Bool("R", false, "copy directories recursively")
	preserve := fs.Bool("p", false, "preserve file attributes")
	force := fs.Bool("f", false, "if an existing destination file cannot be opened, remove it and try again")
	interactive := fs.Bool("i", false, "prompt before overwrite")

	if err := fs.Parse(args); err != nil {
		return 1
	}

	targets := fs.Args()
	if len(targets) < 2 {
		fmt.Fprintln(stderr, "cp: missing file operand")
		return 1
	}

	dest := targets[len(targets)-1]
	sources := targets[:len(targets)-1]

	destInfo, destErr := os.Stat(dest)
	isDestDir := destErr == nil && destInfo.IsDir()

	if len(sources) > 1 && !isDestDir {
		fmt.Fprintf(stderr, "cp: target '%s' is not a directory\n", dest)
		return 1
	}

	exitCode := 0
	reader := bufio.NewReader(stdin)
	isRecursive := *recursive || *recursiveUpper

	for _, src := range sources {
		actualDest := dest
		if isDestDir {
			actualDest = filepath.Join(dest, filepath.Base(src))
		}

		err := doCopy(src, actualDest, isRecursive, *preserve, *force, *interactive, reader, stdout, stderr)
		if err != nil {
			fmt.Fprintf(stderr, "cp: %v\n", err)
			exitCode = 1
		}
	}

	return exitCode
}

func doCopy(src, dst string, recursive, preserve, force, interactive bool, reader *bufio.Reader, stdout, stderr io.Writer) error {
	srcInfo, err := os.Lstat(src)
	if err != nil {
		return err
	}

	if srcInfo.IsDir() {
		if !recursive {
			return fmt.Errorf("-r not specified; omitting directory '%s'", src)
		}
		return copyDir(src, dst, preserve, force, interactive, reader, stdout, stderr)
	}

	// File copy
	if _, err := os.Stat(dst); err == nil {
		if interactive {
			fmt.Fprintf(stdout, "cp: overwrite '%s'? ", dst)
			resp, _ := reader.ReadString('\n')
			resp = strings.ToLower(strings.TrimSpace(resp))
			if resp != "y" && resp != "yes" {
				return nil
			}
		}
		if force {
			os.Remove(dst)
		}
	}

	return copyFileInternal(src, dst, srcInfo, preserve)
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
		os.Chmod(dst, srcInfo.Mode())
		// In a real implementation we would also preserve times, UID, GID etc.
		// For now we preserve mode.
	}
	return nil
}

func copyDir(src, dst string, preserve, force, interactive bool, reader *bufio.Reader, stdout, stderr io.Writer) error {
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
		err = doCopy(filepath.Join(src, entry.Name()), filepath.Join(dst, entry.Name()), true, preserve, force, interactive, reader, stdout, stderr)
		if err != nil {
			return err
		}
	}

	if preserve {
		os.Chmod(dst, srcInfo.Mode())
	}
	return nil
}
