//go:build !linux && !darwin && !windows && !freebsd

package builtin

import "fmt"

func tryCopyOnWrite(src, dst string) (bool, error) {
	return false, fmt.Errorf("reflink not supported on this platform")
}
