package builtin

import (
	"flag"
	"fmt"
	"io"
	"os"
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

	if err := fs.Parse(args); err != nil {
		return 1
	}

	targets := fs.Args()
	if len(targets) == 0 {
		targets = []string{"."}
	}

	exitCode := 0
	for _, target := range targets {
		info, err := os.Stat(target)
		if err != nil {
			fmt.Fprintf(stderr, "ls: %v\n", err)
			exitCode = 1
			continue
		}

		if !info.IsDir() {
			printEntry(stdout, target, info, *long, *human, *classify)
			if !*long {
				fmt.Fprintln(stdout)
			}
			continue
		}

		entries, err := os.ReadDir(target)
		if err != nil {
			fmt.Fprintf(stderr, "ls: %v\n", err)
			exitCode = 1
			continue
		}

		sort.Slice(entries, func(i, j int) bool {
			return entries[i].Name() < entries[j].Name()
		})

		for _, entry := range entries {
			name := entry.Name()
			if !*all && name[0] == '.' {
				continue
			}

			entryInfo, err := entry.Info()
			if err != nil {
				fmt.Fprintf(stderr, "ls: %v\n", err)
				exitCode = 1
				continue
			}

			printEntry(stdout, name, entryInfo, *long, *human, *classify)
			if !*long {
				fmt.Fprint(stdout, "  ")
			}
		}
		if !*long {
			fmt.Fprintln(stdout)
		}
	}

	return exitCode
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
