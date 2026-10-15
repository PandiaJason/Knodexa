//go:build darwin || linux || freebsd || netbsd || openbsd || dragonfly

package storage

import (
	"errors"
	"golang.org/x/sys/unix"
	"os"
)

func tryProcessLock(file *os.File) (bool, error) {
	err := unix.Flock(int(file.Fd()), unix.LOCK_EX|unix.LOCK_NB)
	if errors.Is(err, unix.EWOULDBLOCK) || errors.Is(err, unix.EAGAIN) || errors.Is(err, unix.EINTR) {
		return false, nil
	}
	return err == nil, err
}
func releaseProcessLock(file *os.File) { _ = unix.Flock(int(file.Fd()), unix.LOCK_UN) }
