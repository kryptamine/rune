package stdlib

import (
	stderrors "errors"
	"os"
	"syscall"
	"unsafe"
)

var errLocked = stderrors.New("locked by another process")

var lockFileEx = syscall.NewLazyDLL("kernel32.dll").NewProc("LockFileEx")

const (
	lockfileFailImmediately = 0x1
	lockfileExclusiveLock   = 0x2
	errorLockViolation      = syscall.Errno(33)
)

func lockFile(f *os.File) error {
	var ol syscall.Overlapped

	r, _, err := lockFileEx.Call(f.Fd(), lockfileExclusiveLock|lockfileFailImmediately, 0, 1, 0, uintptr(unsafe.Pointer(&ol)))
	switch {
	case r != 0:
		return nil
	case stderrors.Is(err, errorLockViolation):
		return errLocked
	}

	return err
}
