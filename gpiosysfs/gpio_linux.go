package gpiosysfs

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"unsafe"

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

// Value returns the current value of the pin. True for high and false for low.
func (pin *Pin) Value() (value bool, err error) {
	var buf [1]byte
	_, err = pin.value.ReadAt0(buf[:])
	if err != nil {
		err = wrapPinError(pin, "get value", err)
	}
	value = buf[0] == '1'
	return
}

// SetValue set the current value of the pin. True for high and false for low.
func (pin *Pin) SetValue(value bool) (err error) {
	var buf = [1]byte{'1'}
	if !value {
		buf[0] = '0'
	}
	_, err = pin.value.WriteAt0(buf[:])
	if err != nil {
		err = wrapPinError(pin, "set value", err)
	}
	return
}

// ActiveLow returns whether pin values are inverted for both reading and writing.
func (pin *Pin) ActiveLow(value bool, err error) {
	var buf [1]byte
	_, err = pin.activeLow.ReadAt0(buf[:])
	if err != nil {
		err = wrapPinError(pin, "get activelow", err)
	}
	value = buf[0] == '1'
	return
}

// SetActiveLow sets whether pin values are inverted for both reading and writing.
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

// Interrupt waits for interrupts generated by signal edge changing
// and the pin to ch.
// The returned ch (if no error) will be closed if pin is closed or context is done.
func (pin *Pin) Interrupt(context context.Context) (ch <-chan *Pin, err error) {
	f, err := unix.Open(pin.value.Path, unix.O_RDONLY, 0)
	if err != nil {
		err = fmt.Errorf("failed to open value file of pin %#v when preparing interrupt: %w", pin.n, err)
	}

	// uint64(1) will be written to contextDoneEventFd after context is done.
	contextDoneEventFd, err := unix.Eventfd(0, 0)
	if err != nil {
		err = fmt.Errorf("failed to call Eventfd when preparing interrupt of pin #%v: %w", pin.n, err)
		return
	}
	epollFd, err := unix.EpollCreate(1)
	if err != nil {
		err = fmt.Errorf("failed to call EpollCreate when preparing interrupt of pin #%v: %w", pin.n, err)
		return
	}

	// epoll_wait both contextDoneEventFd and f
	err = unix.EpollCtl(epollFd, unix.EPOLL_CTL_ADD, contextDoneEventFd, &unix.EpollEvent{
		Events: unix.EPOLLIN | unix.EPOLLPRI | unix.EPOLLERR | unix.EPOLLET,
		Fd:     int32(contextDoneEventFd),
	})
	if err != nil {
		err = fmt.Errorf("failed to call EpollCtrl when preparing interrupt of pin #%v: %w", pin.n, err)
		return
	}

	err = unix.EpollCtl(epollFd, unix.EPOLL_CTL_ADD, f, &unix.EpollEvent{
		Events: unix.EPOLLPRI | unix.EPOLLERR | unix.EPOLLET,
		Fd:     int32(f),
	})
	if err != nil {
		err = fmt.Errorf("failed to call EpollCtrl when preparing interrupt of pin #%v: %w", pin.n, err)
		return
	}

	channel := make(chan *Pin)
	ch = channel
	// Start epoll_wait loop.
	go pin.waitInterruptLoop(epollFd, f, contextDoneEventFd, channel)

	// Watch for context done.
	go func() {
		select {
		case <-context.Done():
		case <-pin.cancelInterrupt:
		}
		// Wakeup contextDoneEventFd by writing uint64(1)
		var one = uint64(1)
		_, err := unix.Write(contextDoneEventFd, (*[8]byte)(unsafe.Pointer(&one))[:])
		if err != nil {
			log.Panic(fmt.Errorf("failed to call Write: %w", err))
		}
	}()

	return
}

func (pin *Pin) waitInterruptLoop(epollFd int, f int, contextDoneEventFd int, c chan *Pin) {
	defer func() {
		close(c)
		err := unix.Close(epollFd)
		if err != nil {
			log.Panic(fmt.Errorf("failed to close epoll fd: %w", err))
		}
		err = unix.Close(contextDoneEventFd)
		if err != nil {
			log.Panic(fmt.Errorf("failed to close event fd: %w", err))
		}
		err = unix.Close(f)
		if err != nil {
			log.Panic(fmt.Errorf("failed to close fd: %w", err))
		}
	}()

	var waitEvent [2]unix.EpollEvent
	for {
		n, err := unix.EpollWait(epollFd, waitEvent[:], -1)
		if err != nil {
			log.Panic(fmt.Errorf("failed to call EpollWait: %w", err))
		}
		log.Printf("EpollWait returns %v %v", n, err)
		for i := 0; i < n; i++ {
			switch waitEvent[i].Fd {
			case int32(f):
				// Interrupt caused by signal edge changing.
				c <- pin
			case int32(contextDoneEventFd):
				// Context is done.
				return
			}
		}
	}
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
