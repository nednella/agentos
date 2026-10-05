package cli

import (
	"os"
	"syscall"
	"unsafe"
)

// stdinIsTerminal says whether a person is at the keyboard: Finder, open and hooks give the
// command no terminal.
func stdinIsTerminal() bool {
	var t syscall.Termios
	_, _, errno := syscall.Syscall(syscall.SYS_IOCTL, os.Stdin.Fd(), syscall.TIOCGETA, uintptr(unsafe.Pointer(&t)))
	return errno == 0
}
