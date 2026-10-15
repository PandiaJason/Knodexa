package storage

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// WithProcessLock serializes cooperating processes using an OS-backed lock.
// The persistent lock file must not be removed: deleting it creates another lock
// identity. The OS releases ownership on process exit, including crashes.
func WithProcessLock(path string, operation func() error) error {
	return withProcessLock(path, 5*time.Second, operation)
}

func withProcessLock(path string, timeout time.Duration, operation func() error) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	if info, err := os.Lstat(path); err == nil && info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("lock cannot follow a symlink: %s", path)
	}
	file, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return err
	}
	defer file.Close()
	deadline := time.Now().Add(timeout)
	for {
		locked, err := tryProcessLock(file)
		if err != nil {
			return err
		}
		if locked {
			defer releaseProcessLock(file)
			return operation()
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("timed out waiting for shared memory lock: %s", path)
		}
		time.Sleep(10 * time.Millisecond)
	}
}
