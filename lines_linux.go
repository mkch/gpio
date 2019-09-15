package gpio

import (
	"context"
	"fmt"
	"io"
	"runtime"
	"unsafe"

	"github.com/mkch/gpio/internal/sys"
	"golang.org/x/sys/unix"
)

// InputLines is a batch of opened GPIO lines used as input.
type InputLines interface {
	io.Closer
	// Value returns the current values of the GPIO lines. 1 (high) or 0 (low).
	Values() (values []byte, err error)
}

// OutputLines is a batch of opened GPIO lines used as output.
type OutputLines interface {
	io.Closer
	// Value returns the current values of the GPIO lines. 1 (high) or 0 (low).
	Values() (values []byte, err error)
	// SetValues sets the values of GPIO lines.
	// Value should be 0 (low) or 1 (high), anything else than 0 will be interpreted as 1 (high).
	// If there are more values than requested lines, the extra values will be discarded. If there are less values,
	// the missing values will be 0.
	SetValues(value []byte) (err error)
}

// InputLine is an opened GPIO line used as input.
type InputLine interface {
	io.Closer
	// Value returns the current value of the GPIO line. 1 (high) or 0 (low).
	Value() (value byte, err error)
}

// OutputLine is an opened GPIO line used as output.
type OutputLine interface {
	io.Closer
	// Value returns the current value of the GPIO line. 1 (high) or 0 (low).
	Value() (value byte, err error)
	// SerValue sets the value of the GPIO line.
	// Value should be 0 (low) or 1 (high), anything else than 0 will be interpreted as 1 (high).
	SetValue(value byte) (err error)
}

// InputLineWithEvent is an opened GPIO input line which can be subscribed
// to observe events.
type InputLineWithEvent interface {
	InputLine
	Subscribe(context.Context) (<-chan Event, error)
}

type lines struct {
	fd       int
	numLines int
}

func (l *lines) Close() (err error) {
	err = unix.Close(l.fd)
	l.fd = -1
	return
}

func (l *lines) Fd() uintptr {
	return uintptr(l.fd)
}

func (l *lines) Values() (values []byte, err error) {
	var arg [64]byte
	err = sys.Ioctl(l.fd, sys.GPIOHANDLE_GET_LINE_VALUES_IOCTL, uintptr(unsafe.Pointer(&arg[0])))
	if err != nil {
		err = fmt.Errorf("get GPIO line values failed: %w", err)
		return
	}
	values = arg[:l.numLines]
	return
}

func (l *lines) Value() (value byte, err error) {
	values, err := l.Values()
	if err != nil {
		return
	}
	value = values[0]
	return
}

func (l *lines) SetValues(values []byte) (err error) {
	if len(values) > 64 {
		err = fmt.Errorf("set GPIO line values failed: length of values(%v) > 64", len(values))
	}
	var arg [64]byte
	copy(arg[:], values)
	err = sys.Ioctl(l.fd, sys.GPIOHANDLE_SET_LINE_VALUES_IOCTL, uintptr(unsafe.Pointer(&arg[0])))
	runtime.KeepAlive(arg)
	if err != nil {
		err = fmt.Errorf("set GPIO line values failed: %w", err)
		return
	}
	return
}

func (l *lines) SetValue(value byte) (err error) {
	var values = [1]byte{value}
	err = l.SetValues(values[:])
	runtime.KeepAlive(values)
	return
}
