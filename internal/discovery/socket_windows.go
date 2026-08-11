//go:build windows

package discovery

import (
	"syscall"
)

func setReusePort(fd uintptr) error {
	return syscall.SetsockoptInt(syscall.Handle(fd), syscall.SOL_SOCKET, syscall.SO_REUSEADDR, 1)
}