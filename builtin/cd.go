package builtin

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

func init() {
	RegisterBuiltin(&BuiltinCommand{
		Name:        "cd",
		Description: "切换工作目录",
		Usage:       "cd [-L|-P] [dir]",
		Help:        "切换当前工作目录；不带参数时跳转到 HOME，`cd -` 跳回 OLDPWD。",
		Action:      Cd,
	})
	SetArgCompleter("cd", completeDirectoryPaths)
}

func Cd(args []string, env Environment, stdin io.Reader, stdout io.Writer, stderr io.Writer) int {
	args = PreprocessArgs(args)
	fs := flag.NewFlagSet("cd", flag.ContinueOnError)
	fs.SetOutput(stderr)

	m := RegisterMeta("cd")
	BoolFlag(fs, m, "L", "L", true, "handle the directory operand dot-dot component logically")
	physical := BoolFlag(fs, m, "P", "P", false, "handle the directory operand dot-dot component physically")

	if err := fs.Parse(args); err != nil {
		return 1
	}

	usePhysical := *physical
	for _, arg := range args {
		if arg == "-L" {
			usePhysical = false
		} else if arg == "-P" {
			usePhysical = true
		}
	}

	var dir string
	remaining := fs.Args()
	if len(remaining) == 0 {
		dir = os.Getenv("HOME")
		if dir == "" {
			fmt.Fprintln(stderr, "cd: HOME not set")
			return 1
		}
	} else {
		dir = remaining[0]
	}

	if dir == "-" {
		oldpwd, ok := env.GetString("OLDPWD")
		if !ok {
			fmt.Fprintln(stderr, "cd: OLDPWD not set")
			return 1
		}
		dir = oldpwd
		fmt.Fprintln(stdout, dir)
	}

	// Capture current cwd for OLDPWD and logical path computation.
	curDir, err := os.Getwd()
	if err == nil {
		env.SetString("OLDPWD", curDir)
	}

	// In logical mode (-L, default), compute target path BEFORE os.Chdir.
	// Otherwise relative paths like ".." would be resolved against the new cwd
	// (after chdir), causing PWD to drift one level too far.
	logicalTarget := ""
	if !usePhysical {
		if filepath.IsAbs(dir) {
			logicalTarget = filepath.Clean(dir)
		} else if curDir != "" {
			logicalTarget = filepath.Clean(filepath.Join(curDir, dir))
		}
	}

	err = os.Chdir(dir)
	if err != nil {
		fmt.Fprintf(stderr, "cd: %v\n", err)
		return 1
	}

	newDir, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(stderr, "cd: %v\n", err)
		return 1
	}
	if !usePhysical {
		if logicalTarget != "" {
			newDir = logicalTarget
		}
	} else {
		physDir, err := filepath.EvalSymlinks(newDir)
		if err == nil {
			newDir = physDir
		}
	}

	env.SetString("PWD", newDir)
	return 0
}
