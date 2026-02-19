package builtin

import (
	"fmt"
	"io"
	"sort"
)

func init() {
	RegisterBuiltin("jobs", JobsCmd)
}

func JobsCmd(args []string, env Environment, stdin io.Reader, stdout io.Writer, stderr io.Writer) int {
	JobsMu.Lock()
	defer JobsMu.Unlock()

	var ids []int
	for id := range Jobs {
		ids = append(ids, id)
	}
	sort.Ints(ids)

	for _, id := range ids {
		job := Jobs[id]
		fmt.Fprintf(stdout, "[%d] %-10s %s\n", job.ID, job.Status, job.Command)
	}

	return 0
}
