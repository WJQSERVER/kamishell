package builtin

import (
	"fmt"
	"io"
	"slices"
	"sort"
	"strings"
	"sync"
)

type Environment interface {
	Set(name string, val any)
	Get(name string) (any, bool)
}

type Inspector interface {
	Inspect() string
}

type BuiltinFunc func(args []string, env Environment, stdin io.Reader, stdout io.Writer, stderr io.Writer) int

// BuiltinCommand represents a builtin tool with metadata.
type BuiltinCommand struct {
	Name        string
	Description string
	Usage       string
	Help        string
	Action      BuiltinFunc
}

var Builtins = map[string]*BuiltinCommand{}

// RegisterBuiltin adds a new builtin command to the registry.
func RegisterBuiltin(cmd *BuiltinCommand) {
	Builtins[cmd.Name] = cmd
}

func BuiltinHelpRequested(args []string) bool {
	return slices.Contains(args, "--help")
}

func HandleBuiltinHelp(cmd *BuiltinCommand, args []string, stdout io.Writer) bool {
	if !BuiltinHelpRequested(args) {
		return false
	}
	PrintBuiltinHelp(cmd, stdout)
	return true
}

func PrintBuiltinHelp(cmd *BuiltinCommand, stdout io.Writer) {
	if cmd == nil {
		return
	}

	usage := strings.TrimSpace(cmd.Usage)
	if usage == "" {
		usage = cmd.Name
	}

	fmt.Fprintf(stdout, "用法: %s\n", usage)
	if cmd.Description != "" {
		fmt.Fprintf(stdout, "描述: %s\n", cmd.Description)
	}

	help := strings.TrimSpace(cmd.Help)
	if help != "" {
		fmt.Fprintln(stdout, "")
		fmt.Fprintln(stdout, help)
	}
}

func BuiltinNames() []string {
	names := make([]string, 0, len(Builtins))
	for name := range Builtins {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

type Job struct {
	ID      int
	Command string
	Status  string
	Error   string
}

var (
	Jobs      = make(map[int]*Job)
	NextJobID = 1
	JobsMu    sync.Mutex
)

func RegisterJob(cmd string) int {
	JobsMu.Lock()
	defer JobsMu.Unlock()
	id := NextJobID
	Jobs[id] = &Job{
		ID:      id,
		Command: cmd,
		Status:  "Running",
	}
	NextJobID++
	return id
}

func CompleteJob(id int) {
	CompleteJobWithResult(id, true, "")
}

func CompleteJobWithResult(id int, success bool, errMsg string) {
	JobsMu.Lock()
	defer JobsMu.Unlock()
	if job, ok := Jobs[id]; ok {
		if success {
			job.Status = "Done"
			job.Error = ""
		} else {
			job.Status = "Failed"
			job.Error = errMsg
		}
	}
}

func PreprocessArgs(args []string) []string {
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
