package gpio

import (
	"context"
	"fmt"
	"log"
	"syscall"
	"time"
	"unsafe"

	"github.com/mkch/gpio/internal/sys"
	"golang.org/x/sys/unix"
)

type inputLineWithEvent struct {
	lines

	wakeUpEventFd     int
	wakeUpDataChannel chan *wakeUpData
	closed            chan struct{}
}

func (l *inputLineWithEvent) Close() (err error) {
	err = l.lines.Close()
	if err != nil {
		return
	}
	close(l.closed)
	return
}

func newInputLineEvents(chipFd int, offset uint32, flags, eventFlags uint32, consumer string) (line *inputLineWithEvent, err error) {
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
		Events: unix.EPOLLIN | unix.EPOLLPRI,
		Fd:     int32(req.Fd),
	})
	if err != nil {
		err = fmt.Errorf("request GPIO event failed: epoll_ctl %w", err)
		return
	}

	line = &inputLineWithEvent{
		lines:             lines{fd: int(req.Fd), numLines: 1},
		wakeUpEventFd:     wakeUpEventFd,
		wakeUpDataChannel: make(chan *wakeUpData),
		closed:            make(chan struct{}),
	}

	go line.waitLoop(epollFd)

	go func() {
		<-line.closed
		// Exit wait loop.
		line.wakeUpEpollWaitLoop(nil)
	}()
	return
}

type wakeUpData struct {
	Add           bool
	EventReceiver chan Event
}

func (l *inputLineWithEvent) waitLoop(epollFd int) {
	var eventReceivers []chan Event

	defer func() {
		err := unix.Close(l.wakeUpEventFd)
		if err != nil {
			log.Panic(fmt.Errorf("failed to call close: %w", err))
		}
		err = unix.Close(epollFd)
		if err != nil {
			log.Panic(fmt.Errorf("failed to call close: %w", err))
		}
		for _, receiver := range eventReceivers {
			close(receiver)
		}
	}()

	var waitEvent [2]unix.EpollEvent
	for {
		n, err := unix.EpollWait(epollFd, waitEvent[:], -1)
		if err != nil {
			if err == syscall.EINTR {
				continue
			}
			log.Panic(fmt.Errorf("failed to wait GPIO event: %w", err))
		}
		for i := 0; i < n; i++ {
			switch waitEvent[i].Fd {
			case int32(l.fd):
				// Interrupt caused by GPIO event.
				var eventData sys.GPIOEventData
				n, err := unix.Read(l.fd, (*[unsafe.Sizeof(eventData)]byte)(unsafe.Pointer(&eventData))[:])
				if err != nil {
					if err == syscall.EINTR {
						continue
					}
					log.Panic(fmt.Errorf("failed to read GPIO event: %w", err))
				}
				if n != int(unsafe.Sizeof(eventData)) {
					log.Panic(fmt.Errorf("failed to read GPIO event: short read. %v of %v", n, unsafe.Sizeof(eventData)))
				}

				sec := uint64(time.Nanosecond) * eventData.Timestamp / uint64(time.Second)
				nano := uint64(time.Nanosecond) * eventData.Timestamp % uint64(time.Second)
				event := Event{
					RisingEdge: eventData.ID == sys.GPIOEVENT_EVENT_RISING_EDGE,
					Time:       time.Unix(int64(sec), int64(nano)),
				}
				for _, recv := range eventReceivers {
					recv <- event
				}
			case int32(l.wakeUpEventFd):
				var count uint64
				n, err := unix.Read(l.wakeUpEventFd, (*[unsafe.Sizeof(count)]byte)(unsafe.Pointer(&count))[:])
				if err != nil {
					if err == syscall.EINTR {
						continue
					}
					log.Panic(fmt.Errorf("failed to read event: %w", err))
				}
				if n != int(unsafe.Sizeof(count)) {
					log.Panic(fmt.Errorf("failed to event: short read. %v of %v", n, unsafe.Sizeof(count)))
				}
				for ; count > 0; count-- {
					wakeUpData := <-l.wakeUpDataChannel
					if wakeUpData == nil { // A sign to exit.
						return
					}
					if wakeUpData.Add {
						eventReceivers = append(eventReceivers, wakeUpData.EventReceiver)
					} else {
						for i, recv := range eventReceivers {
							if recv == wakeUpData.EventReceiver {
								close(recv)
								eventReceivers = append(eventReceivers[:i], eventReceivers[i+1:]...)
								break
							}
						}
					}
				}
			}
		}
	}
}

func (l *inputLineWithEvent) wakeUpEpollWaitLoop(data *wakeUpData) (err error) {
	// Wakeup epoll_wait loop adding 1 to the event counter.
	var one = uint64(1)
	n, err := unix.Write(l.wakeUpEventFd, (*[unsafe.Sizeof(one)]byte)(unsafe.Pointer(&one))[:])
	if err != nil {
		err = fmt.Errorf("failed to write to event fd: %w", err)
		return
	}
	if n != int(unsafe.Sizeof(one)) {
		err = fmt.Errorf("failed to write to event fd: short write: %v out of %v", n, unsafe.Sizeof(one))
	}
	l.wakeUpDataChannel <- data
	return
}

func (l *inputLineWithEvent) Subscribe(context context.Context) (events <-chan Event, err error) {
	receiver := make(chan Event, 32)
	events = receiver

	// Add receiver.
	err = l.wakeUpEpollWaitLoop(&wakeUpData{Add: true, EventReceiver: receiver})
	if err != nil {
		return
	}

	go func() {
		select {
		case <-context.Done():
			// Remove receiver.
			l.wakeUpEpollWaitLoop(&wakeUpData{Add: false, EventReceiver: receiver})
		}
	}()

	return
}
