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
	Register("ls", Ls)
}

func Ls(args []string, stdout io.Writer, stderr io.Writer) int {
	args = preprocessArgs(args)

	fs := flag.NewFlagSet("ls", flag.ContinueOnError)
	fs.SetOutput(stderr)

	all := fs.Bool("a", false, "do not ignore entries starting with .")
	long := fs.Bool("l", false, "use a long listing format")
	human := fs.Bool("h", false, "with -l, print sizes like 1K 234M 2G etc.")
	classify := fs.Bool("F", false, "append indicator (one of */=>@|) to entries")

	if err := fs.Parse(args); err != nil {
		return 1
	}

	dirs := fs.Args()
	if len(dirs) == 0 {
		dirs = []string{"."}
	}

	for _, dir := range dirs {
		entries, err := os.ReadDir(dir)
		if err != nil {
			fmt.Fprintf(stderr, "ls: %v\n", err)
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

			if *long {
				info, err := entry.Info()
				if err != nil {
					fmt.Fprintf(stderr, "ls: %v\n", err)
					continue
				}

				mode := info.Mode().String()
				size := formatSize(info.Size(), *human)
				mtime := info.ModTime().Format(time.Stamp)

				fmt.Fprintf(stdout, "%-12s %10s %s %s", mode, size, mtime, name)
			} else {
				fmt.Fprint(stdout, name)
			}

			if *classify {
				fmt.Fprint(stdout, getClassifyIndicator(entry))
			}

			if *long {
				fmt.Fprintln(stdout)
			} else {
				fmt.Fprint(stdout, "  ")
			}
		}
		if !*long {
			fmt.Fprintln(stdout)
		}
	}

	return 0
}

func preprocessArgs(args []string) []string {
	var result []string
	for _, arg := range args {
		if len(arg) > 2 && arg[0] == '-' && arg[1] != '-' {
			for i := 1; i < len(arg); i++ {
				result = append(result, "-"+string(arg[i]))
			}
		} else {
			result = append(result, arg)
		}
	}
	return result
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

func getClassifyIndicator(entry os.DirEntry) string {
	if entry.IsDir() {
		return "/"
	}
	info, err := entry.Info()
	if err != nil {
		return ""
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
