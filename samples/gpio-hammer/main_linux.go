// https://github.com/torvalds/linux/blob/master/tools/gpio/gpio-hammer.c
//
package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/mkch/gpio"
)

func main() {
	flag.Usage = func() {
		fmt.Fprintln(flag.CommandLine.Output(), `Usage: gpio-hammer [options]...
Hammer GPIO lines, 0->1->0->1...`)
		flag.PrintDefaults()
		fmt.Fprintln(flag.CommandLine.Output(), `Example:
gpio-hammer -n gpiochip0 -o 4`)
	}

	deviceName := flag.String("n", "", "Hammer GPIOs on a `name`d device (must be stated)")
	var offsets offsetFlag
	flag.Var(&offsets, "o", "The `offset`[s] to hammer, at least one, several can be stated")
	loops := flag.Uint("c", 0, "Do <`n`> loops (optional, infinite loop if not stated)")
	flag.Parse()

	if len(*deviceName) == 0 || len(offsets) == 0 {
		flag.Usage()
		os.Exit(-1)
	}
	err := hammerDevice(*deviceName, offsets, *loops)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		var errno syscall.Errno
		if errors.As(err, &errno) {
			os.Exit(-int(errno))
		}
		os.Exit(-1)
	}
}

func hammerDevice(deviceName string, offsets []uint32, loops uint) (err error) {
	const swirr = `-\|/`

	chip, err := gpio.OpenChip(deviceName)
	if err != nil {
		return
	}

	var values = make([]byte, len(offsets))
	lines, err := chip.OpenLines(offsets, values, gpio.Output, "gpio-hammer")
	if err != nil {
		return
	}

	initValues, err := lines.Values()
	if err != nil {
		return
	}

	fmt.Printf("Hammer lines %v on %v, initial states: %v\n", offsets, deviceName, initValues)

	// Hammertime!
	var j = 0
	var iter = uint(0)
	for {
		/* Invert all lines so we blink */
		for i := range values {
			if values[i] == 0 {
				values[i] = 1
			} else {
				values[i] = 0
			}
		}

		err = lines.SetValues(values)
		if err != nil {
			return
		}

		/* Re-read values to get status */
		values, err = lines.Values()
		if err != nil {
			return
		}

		fmt.Printf("[%v]", string(swirr[j]))
		j++
		if j == len(swirr) {
			j = 0
		}

		fmt.Printf("[")
		for i := range values {
			fmt.Printf("%v: %v", offsets[i], values[i])
			if i != len(values)-1 {
				fmt.Print(", ")
			}
		}
		fmt.Print("]\r")
		os.Stdin.Sync()

		time.Sleep(time.Second)
		iter++
		if iter == loops {
			break
		}
	}
	return
}

type offsetFlag []uint32

func (f offsetFlag) String() string {
	var s = make([]string, len(f))
	for i, v := range f {
		s[i] = strconv.Itoa(int(v))
	}
	return strings.Join(s, ",")
}

func (f *offsetFlag) Set(str string) (err error) {
	v, err := strconv.ParseUint(str, 10, 32)
	if err != nil {
		return
	}
	*f = append(*f, uint32(v))
	return
}
