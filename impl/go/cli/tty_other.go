//go:build !linux && !darwin && !dragonfly && !freebsd && !netbsd && !openbsd && !windows

package cli

import "os"

func isTerminalFile(f *os.File) bool {
	info, err := f.Stat()
	return err == nil && info.Mode()&os.ModeCharDevice != 0
}
