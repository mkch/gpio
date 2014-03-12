package gpio

import "io/ioutil"
import "os"
import "fmt"
import "strconv"
import "strings"

type Controller struct {
	Base  int    // same as N, the first GPIO managed by this chip.
	Label string // provided for diagnostics (not always unique)
	Ngpio int    // how many GPIOs this manges (N to N + ngpio - 1)
}

const gpio_path = "/sys/class/gpio/"

func Controllers() (ctrls []*Controller, err error) {
	var dir *os.File
	if dir, err = os.Open(gpio_path); err != nil {
		return
	}
	var children []os.FileInfo
	if children, err = dir.Readdir(-1); err != nil {
		return
	}

	for _, child := range children {
		if strings.HasPrefix(child.Name(), "gpiochip") {
			var ctrl *Controller
			if ctrl, err = getController(gpio_path + child.Name() + "/"); err != nil {
				return
			}
			ctrls = append(ctrls, ctrl)
		}
	}
	return
}

func GetController(n int) (*Controller, error) {
	return getController(fmt.Sprintf(gpio_path+"gpiochip%d/", n))
}

func getController(chipDir string) (*Controller, error) {
	var chip Controller
	var buf []byte
	var err error
	if buf, err = ioutil.ReadFile(chipDir + "base"); err != nil {
		return nil, err
	} else if chip.Base, err = strconv.Atoi(trimNewlines(string(buf))); err != nil {
		return nil, err
	}

	if buf, err = ioutil.ReadFile(chipDir + "label"); err != nil {
		return nil, err
	} else {
		chip.Label = trimNewlines(string(buf))
	}

	if buf, err = ioutil.ReadFile(chipDir + "ngpio"); err != nil {
		return nil, err
	} else if chip.Ngpio, err = strconv.Atoi(trimNewlines(string(buf))); err != nil {
		return nil, err
	}

	return &chip, nil
}

type Direction string

const (
	In      Direction = "in"   // RW. The pin is configured as input.
	Out               = "out"  // RW. The pin is configured as output, usually initialized to low.
	OutLow            = "low"  // W. Configure the pin as output and initialize it to low.
	OutHigh           = "high" // W. Configure the pin as output and initialize it to high.
)

type Edge string

const (
	None    Edge = "none" // No edge is selected to generate an interrupt.
	Rising       = "rising"
	Falling      = "falling"
	Both         = "both" // Both rising and falling edges are selected to generate interrupts.
)

const export_path = "/sys/class/gpio/export"

func Export(n int) error {
	return writeExisting(export_path, strconv.Itoa(n))
}

const unexport_path = "/sys/class/gpio/unexport"

func Unexport(n int) error {
	return writeExisting(unexport_path, strconv.Itoa(n))
}

const direction_path_format = "/sys/class/gpio/gpio%d/direction"

func SetDirection(n int, direction Direction) error {
	return writeExisting(fmt.Sprintf(direction_path_format, n), string(direction))
}

func GetDirection(n int) (Direction, error) {
	if content, err := ioutil.ReadFile(fmt.Sprintf(direction_path_format, n)); err != nil {
		return "", err
	} else {
		return Direction(string(content)), nil
	}
}

const edge_path_format = "/sys/class/gpio/gpio%d/edge"

func SetEdge(n int, edge Edge) error {
	return writeExisting(fmt.Sprintf(edge_path_format, n), string(edge))
}

func GetEdge(n int) (Edge, error) {
	if content, err := ioutil.ReadFile(fmt.Sprintf(edge_path_format, n)); err != nil {
		return "", err
	} else {
		return Edge(string(content)), nil
	}
}

const value_path_format = "/sys/class/gpio/gpio%d/value"

func Read(n int) (value int, err error) {
	var content []byte
	if content, err = ioutil.ReadFile(fmt.Sprintf(value_path_format, n)); err != nil {
		return
	}
	return strconv.Atoi(trimNewlines(string(content)))
}

func WaitForInterrupt(n int) (value int, err error) {
	var f *os.File
	if f, err = os.Open(fmt.Sprintf(value_path_format, n)); err != nil {
		return
	}
	defer func() { err = f.Close() }()
	if err = selectPinFd(f.Fd(), 0); err != nil {
		return
	}
	var content []byte
	if content, err = ioutil.ReadAll(f); err != nil {
		return
	}
	return strconv.Atoi(trimNewlines(string(content)))
}

func Write(n int, value int) error {
	return writeExisting(fmt.Sprintf(value_path_format, n), strconv.Itoa(value))
}

const activt_low_path_format = "/sys/class/gpio/gpio%d/active_low"

// Active-low means inverting the logic of the value pin for both reading and writing
// so that a high == 0 and low == 1.
// 0 = false, nonzero = true.
func GetActiveLow(n int) (int, error) {
	if content, err := ioutil.ReadFile(fmt.Sprintf(activt_low_path_format, n)); err != nil {
		return 0, err
	} else {
		return strconv.Atoi(trimNewlines(string(content)))
	}
}

func SetActiveLow(n int, value int) error {
	return writeExisting(fmt.Sprintf(activt_low_path_format, n), strconv.Itoa(value))
}

func writeExisting(name string, content string) error {
	if f, err := os.OpenFile(name, os.O_WRONLY, 0); err != nil {
		return err
	} else {
		defer f.Close()
		if _, err = f.Write([]byte(content)); err != nil {
			return err
		}
	}
	return nil
}

func trimNewlines(str string) string {
	return strings.Trim(str, "\r\n")
}
