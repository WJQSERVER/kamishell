package builtin

import (
	"io"
	"sync"
)

type Environment interface {
	Set(name string, val interface{})
	Get(name string) (interface{}, bool)
}

type Inspector interface {
	Inspect() string
}

type BuiltinFunc func(args []string, env Environment, stdin io.Reader, stdout io.Writer, stderr io.Writer) int

var Builtins = map[string]BuiltinFunc{}

func RegisterBuiltin(name string, fn BuiltinFunc) {
	Builtins[name] = fn
}

type Job struct {
	ID      int
	Command string
	Status  string
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
	JobsMu.Lock()
	defer JobsMu.Unlock()
	if job, ok := Jobs[id]; ok {
		job.Status = "Done"
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
