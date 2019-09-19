package gpiosysfs

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/mkch/gpio/internal/fdevents"
	"golang.org/x/sys/unix"
)

// Chip is the information of a GPIO controller chip.
type Chip struct {
	Base  int    // The first GPIO managed by this chip.
	Label string // The Label fo this chip. Provided for diagnostics (not always unique)
	Ngpio int    // How many GPIOs this chip manges. The GPIOs managed by this chip are in the range of Base to Base + ngpio - 1.
}

const gpioPath = "/sys/class/gpio"

// Controllers returns all GPIO controllers available.
func Controllers() (chips []Chip, err error) {
	dir, err := os.Open(gpioPath)
	if err != nil {
		return
	}
	children, err := dir.Readdir(-1)
	if err != nil {
		return
	}

	chips = make([]Chip, 0, len(children))
	for _, child := range children {
		if strings.HasPrefix(child.Name(), "gpiochip") {
			var chip Chip
			chip, err = newController(filepath.Join(gpioPath, child.Name()))
			if err != nil {
				return
			}
			chips = append(chips, chip)
		}
	}
	return
}

// Controller returns the GPIO controller #n.
func Controller(n int) (Chip, error) {
	filepath.Join()
	var gpiochip = []byte("gpiochip  ")[:len("gpiochip")]
	gpiochipN := strconv.AppendInt(gpiochip, int64(n), 10)
	return newController(filepath.Join(gpioPath, string(gpiochipN)))
}

func newController(chipDir string) (chip Chip, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("failed to get controller: %w", err)
		}
	}()
	buf, err := ioutil.ReadFile(filepath.Join(chipDir, "base"))
	if err != nil {
		return
	}
	chip.Base, err = strconv.Atoi(trimNewlines(buf))
	if err != nil {
		return
	}

	buf, err = ioutil.ReadFile(filepath.Join(chipDir, "label"))
	if err != nil {
		return
	}
	chip.Label = trimNewlines(buf)

	buf, err = ioutil.ReadFile(chipDir + "ngpio")
	if err != nil {
		return
	}
	chip.Ngpio, err = strconv.Atoi(trimNewlines(buf))
	if err != nil {
		return
	}

	return
}

// Pin is a GPIO pin.
type Pin struct {
	n                                 int
	value, direction, edge, activeLow *gpioFile

	cancelInterrupt chan struct{}
}

// OpenPin opens the GPIO pin #n for IO.
func OpenPin(n int) (pin *Pin, err error) {
	dir := fmt.Sprintf("/sys/class/gpio/gpio%d", n)
	fi, err := os.Stat(dir)
	if err != nil {
		if !os.IsNotExist(err) {
			err = fmt.Errorf("failed to open pin #%v: %w", n, err)
			return
		}
		const exportPath = "/sys/class/gpio/export"
		err = writeExisting(exportPath, strconv.Itoa(n))
		if err != nil {
			err = fmt.Errorf("failed to open pin #%v: %w", n, err)
			return
		}
	} else if !fi.IsDir() {
		err = fmt.Errorf("failed to open pin #%v: %v is not a dir", n, dir)
		return
	}

	pin = &Pin{n: n,
		cancelInterrupt: make(chan struct{}),
		direction:       &gpioFile{Path: filepath.Join(dir, "direction")},
		value:           &gpioFile{Path: filepath.Join(dir, "value")},
		edge:            &gpioFile{Path: filepath.Join(dir, "edge")},
		activeLow:       &gpioFile{Path: filepath.Join(dir, "active_low")},
	}
	return
}

// Close closes the pin.
func (pin *Pin) Close() (err error) {
	close(pin.cancelInterrupt)
	err = writeExisting("/sys/class/gpio/unexport", strconv.Itoa(pin.n))
	if err != nil {
		err = fmt.Errorf("failed to close pin #%v: %w", pin.n, err)
	}

	if pin.value != nil {
		err = pin.value.Close()
		if err != nil {
			err = fmt.Errorf("failed to close pin #%v: %w", pin.n, err)
			return
		}
	}
	if pin.direction != nil {
		err = pin.direction.Close()
		if err != nil {
			err = fmt.Errorf("failed to close pin #%v: %w", pin.n, err)
			return
		}
	}
	if pin.edge != nil {
		err = pin.edge.Close()
		if err != nil {
			err = fmt.Errorf("failed to close pin #%v: %w", pin.n, err)
			return
		}
	}
	if pin.activeLow != nil {
		err = pin.activeLow.Close()
		if err != nil {
			err = fmt.Errorf("failed to close pin #%v: %w", pin.n, err)
			return
		}
	}
	return
}

// Direction is the IO direction.
type Direction string

// Available directions.
const (
	In      Direction = "in"   // RW. The pin is configured as input.
	Out               = "out"  // RW. The pin is configured as output, usually initialized to low.
	OutLow            = "low"  // W. Configure the pin as output and initialize it to low.
	OutHigh           = "high" // W. Configure the pin as output and initialize it to high.
)

// SetDirection sets the IO direction of the pin.
// pin.SetDirection(Out) may fail if Edge is not None.
func (pin *Pin) SetDirection(direction Direction) (err error) {
	_, err = pin.direction.WriteAt0([]byte(direction))
	if err != nil {
		err = wrapPinError(pin, "set direction", err)
	}
	return
}

// Must be greater or equal to the max length of Edge and Direction constants.
const strBufLen = 16

// Direction returns the IO direction of the pin.
// The return values is In or Out.
func (pin *Pin) Direction() (dir Direction, err error) {
	var buf [strBufLen]byte
	n, err := pin.direction.ReadAt0(buf[:])
	if err != nil {
		if err != io.EOF {
			err = wrapPinError(pin, "get direction", err)
			return
		}
	}
	dir = Direction(trimNewlines(buf[:n]))
	return
}

// Edge is the signal edge that will make Interrupt send value to the channel.
type Edge string

const (
	// None means no edge is selected to generate interrupts.
	None Edge = "none"
	// Rising edges is is selected to generate interrupts. Rising: level is getting to high from low.
	Rising = "rising"
	// Falling edges is is selected to generate interrupts. Falling: level is getting to low from hight.
	Falling = "falling"
	// Both rising and falling edges are selected to generate interrupts.
	Both = "both"
)

// SetEdge sets which edges are selected to generate interrupts.
// Not all GPIO pins are configured to support edge selection,
// so, Edge should be called to confirm the desired edge are set actually.
func (pin *Pin) SetEdge(edge Edge) (err error) {
	_, err = pin.edge.WriteAt0([]byte(edge))
	if err != nil {
		err = wrapPinError(pin, "set edge", err)
	}
	return
}

// Edge returns which edges are selected to generate interrupts.
func (pin *Pin) Edge() (edge Edge, err error) {
	var buf [strBufLen]byte
	n, err := pin.edge.ReadAt0(buf[:])
	if err != nil {
		if err != io.EOF {
			err = wrapPinError(pin, "get edge", err)
			return
		}
	}
	edge = Edge(trimNewlines(buf[:n]))
	return
}

// Value returns the current value of the pin. 1 for high and 0 for low.
func (pin *Pin) Value() (value byte, err error) {
	var buf [1]byte
	_, err = pin.value.ReadAt0(buf[:])
	if err != nil {
		err = wrapPinError(pin, "get value", err)
	}
	if buf[0] == '0' {
		value = 0
	} else {
		value = 1
	}
	return
}

// SetValue set the current value of the pin. 1 for high and 0 for low.
func (pin *Pin) SetValue(value byte) (err error) {
	var buf = [1]byte{'1'}
	if value == 0 {
		buf[0] = '0'
	}
	_, err = pin.value.WriteAt0(buf[:])
	if err != nil {
		err = wrapPinError(pin, "set value", err)
	}
	return
}

// ActiveLow returns whether the pin is configured as active low.
func (pin *Pin) ActiveLow(value bool, err error) {
	var buf [1]byte
	_, err = pin.activeLow.ReadAt0(buf[:])
	if err != nil {
		err = wrapPinError(pin, "get activelow", err)
	}
	value = buf[0] == '1'
	return
}

// SetActiveLow sets whether pin is configured as active low.
func (pin *Pin) SetActiveLow(value bool) (err error) {
	var buf = [1]byte{'1'}
	if !value {
		buf[0] = '0'
	}
	_, err = pin.activeLow.WriteAt0(buf[:])
	if err != nil {
		err = wrapPinError(pin, "set activelow", err)
	}
	return
}

// PinWithEvent is an opened GPIO pin whose events can be read.
type PinWithEvent struct {
	*Pin
	events *fdevents.FdEvents
}

// OpenPinWithEvents opens a GPIO pin for input and GPIO events.
func OpenPinWithEvents(n int) (pin *PinWithEvent, err error) {
	p, err := OpenPin(n)
	if err != nil {
		return
	}
	err = p.SetDirection(In)
	if err != nil {
		return
	}

	fd, err := unix.Open(p.value.Path, unix.O_RDONLY, 0)
	if err != nil {
		err = fmt.Errorf("failed to open value file of pin %#v when preparing interrupt: %w", p.n, err)
		return
	}

	events, err := fdevents.New(fd, true, unix.EPOLLPRI|unix.EPOLLERR, func(fd int) *fdevents.Event {
		v, err := p.Value()
		if err != nil {
			panic(fmt.Errorf("failed to read GPIO event: %w", err))
		}
		return &fdevents.Event{RisingEdge: v == 1, Time: time.Now()}
	})
	if err != nil {
		return
	}

	pin = &PinWithEvent{
		Pin:    p,
		events: events,
	}

	return
}

func (pin *PinWithEvent) Close() (err error) {
	// Close pin.events first.
	// pin.Pin is still used by pin.events before pin.events is closed.
	err1 := pin.events.Close()
	err2 := pin.Pin.Close()
	if err1 != nil {
		return err1
	}
	return err2
}

type Event = fdevents.Event

// Events returns a channel from which the occurrence time of GPIO events can be read.
// The GPIO events of this pin will be sent to the returned channel, and the channel is closed when l is closed.
//
// Package gpiosysfs will not block sending to the channel: it only keeps the lastest
// value in the channel.
func (pin *PinWithEvent) Events() <-chan *Event {
	return pin.events.Events()
}

func writeExisting(path string, content string) (err error) {
	f, err := os.OpenFile(path, os.O_WRONLY, 0)
	if err != nil {
		return
	}
	defer f.Close()
	if _, err = f.Write([]byte(content)); err != nil {
		return err
	}
	return
}

func trimNewlines(str []byte) string {
	return string(bytes.Trim(str, "\r\n"))
}

func wrapPinError(pin *Pin, action string, err error) error {
	return fmt.Errorf("failed to %v of GPIO pin #%v: %w", action, pin.n, err)
}
