package builtin

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"time"
)

func init() {
	RegisterBuiltin("ls", Ls)
}

func Ls(args []string, env Environment, stdin io.Reader, stdout io.Writer, stderr io.Writer) int {
	args = PreprocessArgs(args)

	fs := flag.NewFlagSet("ls", flag.ContinueOnError)
	fs.SetOutput(stderr)

	all := fs.Bool("a", false, "do not ignore entries starting with .")
	long := fs.Bool("l", false, "use a long listing format")
	human := fs.Bool("h", false, "with -l, print sizes like 1K 234M 2G etc.")
	classify := fs.Bool("F", false, "append indicator (one of */=>@|) to entries")
	recursive := fs.Bool("R", false, "list subdirectories recursively")
	reverse := fs.Bool("r", false, "reverse order while sorting")
	sortByTime := fs.Bool("t", false, "sort by modification time, newest first")
	sortBySize := fs.Bool("S", false, "sort by file size, largest first")
	dirOnly := fs.Bool("d", false, "list directories themselves, not their contents")

	if err := fs.Parse(args); err != nil {
		return 1
	}

	targets := fs.Args()
	if len(targets) == 0 {
		targets = []string{"."}
	}

	exitCode := 0
	for i, target := range targets {
		if len(targets) > 1 && !*recursive {
			fmt.Fprintf(stdout, "%s:\n", target)
		}

		info, err := os.Lstat(target)
		if err != nil {
			fmt.Fprintf(stderr, "ls: %v\n", err)
			exitCode = 1
			continue
		}

		if *dirOnly || !info.IsDir() {
			printEntry(stdout, target, info, *long, *human, *classify)
			if !*long {
				fmt.Fprintln(stdout)
			}
		} else {
			ec := listDir(target, stdout, stderr, *all, *long, *human, *classify, *recursive, *reverse, *sortByTime, *sortBySize, len(targets) > 1 || *recursive, i == 0)
			if ec != 0 {
				exitCode = ec
			}
		}

		if i < len(targets)-1 && !*long && !*dirOnly && info.IsDir() {
			fmt.Fprintln(stdout)
		}
	}

	return exitCode
}

func listDir(dirPath string, stdout, stderr io.Writer, all, long, human, classify, recursive, reverse, sortByTime, sortBySize, showHeader, isFirst bool) int {
	if showHeader && !isFirst {
		fmt.Fprintf(stdout, "\n%s:\n", dirPath)
	} else if showHeader && isFirst && recursive {
		fmt.Fprintf(stdout, "%s:\n", dirPath)
	}

	entries, err := os.ReadDir(dirPath)
	if err != nil {
		fmt.Fprintf(stderr, "ls: %v\n", err)
		return 1
	}

	var infos []os.FileInfo
	for _, entry := range entries {
		name := entry.Name()
		if !all && name[0] == '.' {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			continue
		}
		infos = append(infos, info)
	}

	// Sorting
	sort.Slice(infos, func(i, j int) bool {
		var res bool
		if sortByTime {
			res = infos[i].ModTime().After(infos[j].ModTime())
		} else if sortBySize {
			res = infos[i].Size() > infos[j].Size()
		} else {
			res = infos[i].Name() < infos[j].Name()
		}
		if reverse {
			return !res
		}
		return res
	})

	for _, info := range infos {
		printEntry(stdout, info.Name(), info, long, human, classify)
		if !long {
			fmt.Fprint(stdout, "  ")
		}
	}
	if !long {
		fmt.Fprintln(stdout)
	}

	if recursive {
		for _, info := range infos {
			if info.IsDir() && info.Name() != "." && info.Name() != ".." {
				listDir(filepath.Join(dirPath, info.Name()), stdout, stderr, all, long, human, classify, recursive, reverse, sortByTime, sortBySize, true, false)
			}
		}
	}

	return 0
}

func printEntry(stdout io.Writer, name string, info os.FileInfo, long, human, classify bool) {
	if long {
		mode := info.Mode().String()
		size := formatSize(info.Size(), human)
		mtime := info.ModTime().Format(time.Stamp)
		fmt.Fprintf(stdout, "%-12s %10s %s %s", mode, size, mtime, name)
	} else {
		fmt.Fprint(stdout, name)
	}

	if classify {
		fmt.Fprint(stdout, getInfoClassifyIndicator(info))
	}

	if long {
		fmt.Fprintln(stdout)
	}
}

func formatSize(size int64, human bool) string {
	if !human {
		return fmt.Sprintf("%d", size)
	}
	const unit = 1024
	if size < unit {
		return fmt.Sprintf("%dB", size)
	}
	div, exp := int64(unit), 0
	for n := size / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f%c", float64(size)/float64(div), "KMGTPE"[exp])
}

func getInfoClassifyIndicator(info os.FileInfo) string {
	if info.IsDir() {
		return "/"
	}
	mode := info.Mode()
	if mode&os.ModeSymlink != 0 {
		return "@"
	}
	if mode&0111 != 0 {
		return "*"
	}
	return ""
}
