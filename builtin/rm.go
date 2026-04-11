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
		Name:        "rm",
		Description: "删除文件或目录",
		Usage:       "rm [-f] [-i] [-r|-R] [-v] [--no-preserve-root] target...",
		Help:        "删除文件或目录；递归删除目录时默认保护根目录。",
		Action:      Rm,
	})
}

func Rm(args []string, env Environment, stdin io.Reader, stdout io.Writer, stderr io.Writer) int {
	args = PreprocessArgs(args)

	fs := flag.NewFlagSet("rm", flag.ContinueOnError)
	fs.SetOutput(stderr)

	force := fs.Bool("f", false, "ignore nonexistent files and arguments, never prompt")
	interactive := fs.Bool("i", false, "prompt before every removal")
	recursive := fs.Bool("r", false, "remove directories and their contents recursively")
	recursiveUpper := fs.Bool("R", false, "remove directories and their contents recursively")
	verbose := fs.Bool("v", false, "explain what is being done")
	noPreserveRoot := fs.Bool("no-preserve-root", false, "do not treat '/' specially")
	_ = fs.Bool("preserve-root", true, "do not remove '/' (default)")

	if err := fs.Parse(args); err != nil {
		return 1
	}

	isRecursive := *recursive || *recursiveUpper
	targets := fs.Args()

	if len(targets) == 0 && !*force {
		fmt.Fprintln(stderr, "rm: missing operand")
		return 1
	}

	exitCode := 0
	reader := bufio.NewReader(stdin)

	for _, target := range targets {
		if target == "." || target == ".." {
			fmt.Fprintf(stderr, "rm: refusing to remove '.' or '..' directory: skipping '%s'\n", target)
			exitCode = 1
			continue
		}

		if isRecursive && !*noPreserveRoot && isRoot(target) {
			fmt.Fprintf(stderr, "rm: it is dangerous to operate recursively on '/'\n")
			fmt.Fprintf(stderr, "rm: use --no-preserve-root to override this failsafe\n")
			exitCode = 1
			continue
		}

		info, err := os.Lstat(target)
		if err != nil {
			if os.IsNotExist(err) {
				if !*force {
					fmt.Fprintf(stderr, "rm: cannot remove '%s': No such file or directory\n", target)
					exitCode = 1
				}
				continue
			}
			fmt.Fprintf(stderr, "rm: cannot remove '%s': %v\n", target, err)
			exitCode = 1
			continue
		}

		if info.IsDir() && !isRecursive {
			fmt.Fprintf(stderr, "rm: cannot remove '%s': Is a directory\n", target)
			exitCode = 1
			continue
		}

		if *interactive {
			fmt.Fprintf(stdout, "rm: remove '%s'? ", target)
			response, readErr := reader.ReadString('\n')
			if readErr != nil {
				fmt.Fprintf(stderr, "rm: failed to read confirmation: %v\n", readErr)
				exitCode = 1
				continue
			}
			response = strings.ToLower(strings.TrimSpace(response))
			if response != "y" && response != "yes" {
				continue
			}
		}

		if info.IsDir() && isRecursive {
			if *interactive {
				err = removeRecursiveInteractive(target, reader, stdout, stderr, *verbose)
			} else {
				if *verbose {
					err = walkAndRemove(target, false, reader, stdout, stderr, *verbose)
				} else {
					err = os.RemoveAll(target)
				}
			}
		} else {
			err = os.Remove(target)
			if err == nil && *verbose {
				fmt.Fprintf(stdout, "removed '%s'\n", target)
			}
		}

		if err != nil {
			fmt.Fprintf(stderr, "rm: cannot remove '%s': %v\n", target, err)
			exitCode = 1
		}
	}

	return exitCode
}

func isRoot(path string) bool {
	abs, err := filepath.Abs(path)
	if err != nil {
		return false
	}
	// On Unix-like, VolumeName is empty and root is "/"
	// On Windows, VolumeName is e.g. "C:" and root is "C:\"
	return abs == filepath.VolumeName(abs)+string(os.PathSeparator)
}

func removeRecursiveInteractive(path string, reader *bufio.Reader, stdout, stderr io.Writer, verbose bool) error {
	return walkAndRemove(path, true, reader, stdout, stderr, verbose)
}

func walkAndRemove(path string, interactive bool, reader *bufio.Reader, stdout, stderr io.Writer, verbose bool) error {
	info, err := os.Lstat(path)
	if err != nil {
		return err
	}

	if info.IsDir() {
		entries, err := os.ReadDir(path)
		if err != nil {
			return err
		}

		if interactive {
			fmt.Fprintf(stdout, "rm: descend into directory '%s'? ", path)
			resp, readErr := reader.ReadString('\n')
			if readErr != nil {
				return fmt.Errorf("rm: failed to read confirmation: %w", readErr)
			}
			resp = strings.ToLower(strings.TrimSpace(resp))
			if resp == "y" || resp == "yes" {
				for _, entry := range entries {
					walkAndRemove(filepath.Join(path, entry.Name()), interactive, reader, stdout, stderr, verbose)
				}
				fmt.Fprintf(stdout, "rm: remove directory '%s'? ", path)
				resp, readErr = reader.ReadString('\n')
				if readErr != nil {
					return fmt.Errorf("rm: failed to read confirmation: %w", readErr)
				}
				resp = strings.ToLower(strings.TrimSpace(resp))
				if resp == "y" || resp == "yes" {
					err = os.Remove(path)
					if err == nil && verbose {
						fmt.Fprintf(stdout, "removed directory '%s'\n", path)
					}
					return err
				}
			}
			return nil
		} else {
			for _, entry := range entries {
				walkAndRemove(filepath.Join(path, entry.Name()), false, reader, stdout, stderr, verbose)
			}
			err = os.Remove(path)
			if err == nil && verbose {
				fmt.Fprintf(stdout, "removed directory '%s'\n", path)
			}
			return err
		}
	} else {
		if interactive {
			fmt.Fprintf(stdout, "rm: remove '%s'? ", path)
			resp, readErr := reader.ReadString('\n')
			if readErr != nil {
				return fmt.Errorf("rm: failed to read confirmation: %w", readErr)
			}
			resp = strings.ToLower(strings.TrimSpace(resp))
			if resp != "y" && resp != "yes" {
				return nil
			}
		}
		err = os.Remove(path)
		if err == nil && verbose {
			fmt.Fprintf(stdout, "removed '%s'\n", path)
		}
		return err
	}
}
