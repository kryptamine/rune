//go:build !windows

package stdlib

import (
	stderrors "errors"
	"os"
	"syscall"
)

var errLocked = stderrors.New("locked by another process")

func lockFile(f *os.File) error {
	err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB)
	if stderrors.Is(err, syscall.EWOULDBLOCK) {
		return errLocked
	}

	return err
}
