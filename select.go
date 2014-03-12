package gpio

import "syscall"
import "time"
import "unsafe"

//The following C code are form http://fxr.watson.org/fxr/source/sys/select.h
//
// typedef unsigned long   __fd_mask;
//
//#define FD_SETSIZE      1024U
//
//
// #define FD_SETSIZE      1024U
// #endif
//
// #define _NFDBITS        (sizeof(__fd_mask) * 8) /* bits per mask */
// #if __BSD_VISIBLE
// #define NFDBITS         _NFDBITS
// #endif
//
// #ifndef _howmany
// #define _howmany(x, y)  (((x) + ((y) - 1)) / (y))
// #endif
//
// typedef struct fd_set {
//         __fd_mask       __fds_bits[_howmany(FD_SETSIZE, _NFDBITS)];
// } fd_set;
//
//
// #define __fdset_mask(n) ((__fd_mask)1 << ((n) % _NFDBITS))
// #define FD_CLR(n, p)    ((p)->__fds_bits[(n)/_NFDBITS] &= ~__fdset_mask(n))
// #if __BSD_VISIBLE
// #define FD_COPY(f, t)   (void)(*(t) = *(f))
// #endif
// #define FD_ISSET(n, p)  (((p)->__fds_bits[(n)/_NFDBITS] & __fdset_mask(n)) != 0)
// #define FD_SET(n, p)    ((p)->__fds_bits[(n)/_NFDBITS] |= __fdset_mask(n))
// #define FD_ZERO(p) do {                                 \
//         fd_set *_p;                                     \
//         __size_t _n;                                    \
//                                                         \
//         _p = (p);                                       \
//         _n = _howmany(FD_SETSIZE, _NFDBITS);            \
//         while (_n > 0)                                  \
//                 _p->__fds_bits[--_n] = 0;               \
// } while (0)
type __fd_mask uintptr

const _FD_SETSIZE uintptr = 1024
const _NFDBITS = unsafe.Sizeof(__fd_mask(0)) * 8
const _howmany = (_FD_SETSIZE + (_NFDBITS - 1)) / _NFDBITS

type _fd_set [_howmany]__fd_mask

func (p *_fd_set) convertToSyscallFdSetPtr() *syscall.FdSet {
	return (*syscall.FdSet)(unsafe.Pointer(&(*p)[0]))
}

func __fdset_mask(n uintptr) __fd_mask {
	return __fd_mask(1) << (n % _NFDBITS)
}

func fd_set(n uintptr, p *_fd_set) {
	p[n/_NFDBITS] |= __fdset_mask(n)
}

func fd_clear(n uintptr, p *_fd_set) {
	p[n/_NFDBITS] &= ^__fdset_mask(n)
}

func fd_isset(n uintptr, p *_fd_set) bool {
	return p[n/_NFDBITS]&__fdset_mask(n) != 0
}

func fd_zero(p *_fd_set) {
	for i := 0; i < len(p); i++ {
		p[i] = 0
	}
}

func selectPinFd(fd uintptr, timeout time.Duration) error {
	if _, err := selectPinFds([]uintptr{fd}, timeout); err != nil {
		return err
	}
	return nil
}

func selectPinFds(fds []uintptr, timeout time.Duration) (fdSetIndexes []int, err error) {
	if len(fds) == 0 {
		return nil, nil
	}

	var t *syscall.Timeval
	if timeout > 0 {
		tv := syscall.NsecToTimeval(int64(timeout))
		t = &tv
	}

	var exceptFdSet _fd_set
	var nfds int
	for _, fd := range fds {
		if int(fd) > nfds {
			nfds = int(fd)
		}
		fd_set(fd, &exceptFdSet)
	}
	nfds++

	var ret int
	if ret, err = syscall_Select(nfds, nil, nil, exceptFdSet.convertToSyscallFdSetPtr(), t); err != nil { // Error occured.
		return
	} else if ret == 0 { // Timeouted.
		return
	} else {
		for i, fd := range fds {
			if fd_isset(fd, &exceptFdSet) {
				fdSetIndexes = append(fdSetIndexes, i)
			}
		}
		return
	}
}
