package gpio

import "syscall"
import "unsafe"

// The return value of syscall.Select() in darwin is missing.
// This is the "full" implementation.
func syscall_Select(nfd int, r *syscall.FdSet, w *syscall.FdSet, e *syscall.FdSet, timeout *syscall.Timeval) (n int, err error) {
	r0, _, e1 := syscall.Syscall6(syscall.SYS_SELECT, uintptr(nfd), uintptr(unsafe.Pointer(r)), uintptr(unsafe.Pointer(w)), uintptr(unsafe.Pointer(e)), uintptr(unsafe.Pointer(timeout)), 0)
	n = int(r0)
	if e1 != 0 {
		err = e1
	}
	return
}
