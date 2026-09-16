package cli

import (
	"os"
	"syscall"
)

func isTerminalFile(f *os.File) bool {
	var mode uint32
	return syscall.GetConsoleMode(syscall.Handle(f.Fd()), &mode) == nil
}
