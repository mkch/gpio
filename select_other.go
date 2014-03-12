// +build !darwin

package gpio

import "syscall"

func syscall_Select(nfd int, r *syscall.FdSet, w *syscall.FdSet, e *syscall.FdSet, timeout *syscall.Timeval) (n int, err error) {
	return syscall.Select(nfd, r, w, e, timeout)
}
