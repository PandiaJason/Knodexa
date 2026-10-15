//go:build !darwin && !linux && !freebsd && !netbsd && !openbsd && !dragonfly && !windows

package storage

import (
	"fmt"
	"os"
)

func tryProcessLock(file *os.File) (bool, error) {
	return false, fmt.Errorf("shared memory locks are unsupported on this platform")
}
func releaseProcessLock(file *os.File) {}
