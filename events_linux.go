package gpio

import (
	"fmt"
	"log"
	"os"
	"syscall"
	"time"
	"unsafe"

	"github.com/mkch/gpio/internal/sys"
	"golang.org/x/sys/unix"
)

var Logger *log.Logger = log.New(os.Stderr, "gpio: ", log.LstdFlags)

// LineWithEvent is an opened GPIO line whose events can be subscribed.
type LineWithEvent struct {
	l Line
	// Discard the edge info of GPIO event. It is not reliably reflect
	// the current state of line if the event consuming is slower than
	// producing.
	events              chan time.Time
	closed              chan struct{}
	exitWaitLoopEventFd int
}

func (l *LineWithEvent) Close() (err error) {
	return l.l.Close()
}

// Value returns the current value of the GPIO line. 1 (high) or 0 (low).
func (l *LineWithEvent) Value() (value byte, err error) {
	return l.l.Value()
}

func newInputLineEvents(chipFd int, offset uint32, flags, eventFlags uint32, consumer string) (line *LineWithEvent, err error) {
	var req = sys.GPIOEventRequest{
		LineOffset:  offset,
		HandleFlags: uint32(flags),
		EventFlags:  uint32(eventFlags)}
	copy(req.ConsumerLabel[:], consumer)
	err = sys.Ioctl(chipFd, sys.GPIO_GET_LINEEVENT_IOCTL, uintptr(unsafe.Pointer(&req)))
	if err != nil {
		err = fmt.Errorf("request GPIO event failed: ioctl %w", err)
		return
	}

	wakeUpEventFd, err := unix.Eventfd(0, 0)
	if err != nil {
		err = fmt.Errorf("request GPIO event failed: eventfd: %w", err)
		return
	}

	epollFd, err := unix.EpollCreate(1)
	if err != nil {
		err = fmt.Errorf("request GPIO event failed: epoll_create: %w", err)
		return
	}

	err = unix.EpollCtl(epollFd, unix.EPOLL_CTL_ADD, wakeUpEventFd, &unix.EpollEvent{
		Events: unix.EPOLLIN,
		Fd:     int32(wakeUpEventFd),
	})
	if err != nil {
		err = fmt.Errorf("request GPIO event failed: epoll_ctl %w", err)
		return
	}

	err = unix.EpollCtl(epollFd, unix.EPOLL_CTL_ADD, int(req.Fd), &unix.EpollEvent{
		// DO NOT use unix.EPOLLET which will cause event lost when BothEdges is set.
		Events: unix.EPOLLIN | unix.EPOLLPRI | unix.EPOLLET,
		Fd:     int32(req.Fd),
	})
	if err != nil {
		err = fmt.Errorf("request GPIO event failed: epoll_ctl %w", err)
		return
	}

	line = &LineWithEvent{
		l:                   Line{fd: int(req.Fd), numLines: 1},
		exitWaitLoopEventFd: wakeUpEventFd,
		closed:              make(chan struct{}),
		events:              make(chan time.Time, 1), // Buffer 1 to store the latest.
	}

	go line.waitLoop(epollFd)

	go func() {
		<-line.closed
		// Exit wait loop.
		line.notifyWaitLoopToExit()
	}()
	return
}

func (l *LineWithEvent) waitLoop(epollFd int) {
	defer func() {
		err := unix.Close(l.exitWaitLoopEventFd)
		if err != nil {
			Logger.Panic(fmt.Errorf("failed to call close: %w", err))
		}
		err = unix.Close(epollFd)
		if err != nil {
			Logger.Panic(fmt.Errorf("failed to call close: %w", err))
		}
		close(l.events)
	}()

	var waitEvent [2]unix.EpollEvent
epoll_wait_loop:
	for {
		n, err := unix.EpollWait(epollFd, waitEvent[:], -1)
		if err != nil {
			if err == syscall.EINTR {
				continue
			}
			Logger.Panic(fmt.Errorf("failed to wait GPIO event: %w", err))
		}
		for i := 0; i < n; i++ {
			switch waitEvent[i].Fd {
			case int32(l.l.fd):
				// Interrupt caused by GPIO event.
				var eventData sys.GPIOEventData
				n, err := unix.Read(l.l.fd, (*[unsafe.Sizeof(eventData)]byte)(unsafe.Pointer(&eventData))[:])
				if err != nil {
					if err == syscall.EINTR {
						continue
					}
					Logger.Panic(fmt.Errorf("failed to read GPIO event: %w", err))
				}
				if n != int(unsafe.Sizeof(eventData)) {
					Logger.Panic(fmt.Errorf("failed to read GPIO event: short read. %v of %v", n, unsafe.Sizeof(eventData)))
				}

				sec := uint64(time.Nanosecond) * eventData.Timestamp / uint64(time.Second)
				nano := uint64(time.Nanosecond) * eventData.Timestamp % uint64(time.Second)
				t := time.Unix(int64(sec), int64(nano))
				// Discard the unread old value.
				select {
				case <-l.events:
				default:
				}
				// Send the latest.
				l.events <- t
			case int32(l.exitWaitLoopEventFd):
				break epoll_wait_loop
			}
		}
	}
}

func (l *LineWithEvent) notifyWaitLoopToExit() (err error) {
	// Wakeup epoll_wait loop adding 1 to the event counter.
	var one = uint64(1)
	n, err := unix.Write(l.exitWaitLoopEventFd, (*[unsafe.Sizeof(one)]byte)(unsafe.Pointer(&one))[:])
	if err != nil {
		err = fmt.Errorf("failed to write to event fd: %w", err)
		return
	}
	if n != int(unsafe.Sizeof(one)) {
		err = fmt.Errorf("failed to write to event fd: short write: %v out of %v", n, unsafe.Sizeof(one))
	}
	return
}

// Events returns an channel from which the occurrence time of GPIO events can be read.
// The best estimate of time of event occurrence is sent to the returned channel,
// and the channel is closed when l is closed.
//
// Package gpio will not block sending to the channel: it only keeps the lastest
// value in the channel.
func (l *LineWithEvent) Events() chan time.Time {
	return l.events
}
