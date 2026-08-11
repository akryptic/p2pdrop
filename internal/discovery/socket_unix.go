//go:build !windows

package discovery

import (
	"golang.org/x/sys/unix"
)

func setReusePort(fd uintptr) error {
	err := unix.SetsockoptInt(int(fd), unix.SOL_SOCKET, unix.SO_REUSEPORT, 1)
	if err != nil {
		return unix.SetsockoptInt(int(fd), unix.SOL_SOCKET, unix.SO_REUSEADDR, 1)
	}
	return nil
}
