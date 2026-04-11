package builtin

import (
	"fmt"
	"io"
	"sort"
)

func init() {
	RegisterBuiltin(&BuiltinCommand{
		Name:        "jobs",
		Description: "列出后台作业",
		Usage:       "jobs",
		Help:        "列出当前 shell 中注册的后台任务及状态。",
		Action:      JobsCmd,
	})
}

func JobsCmd(args []string, env Environment, stdin io.Reader, stdout io.Writer, stderr io.Writer) int {
	if HandleBuiltinHelp(Builtins["jobs"], args, stdout) {
		return 0
	}
	JobsMu.Lock()
	defer JobsMu.Unlock()

	var ids []int
	for id := range Jobs {
		ids = append(ids, id)
	}
	sort.Ints(ids)

	for _, id := range ids {
		job := Jobs[id]
		if job.Error != "" {
			fmt.Fprintf(stdout, "[%d] %-10s %s (%s)\n", job.ID, job.Status, job.Command, job.Error)
			continue
		}
		fmt.Fprintf(stdout, "[%d] %-10s %s\n", job.ID, job.Status, job.Command)
	}

	return 0
}
