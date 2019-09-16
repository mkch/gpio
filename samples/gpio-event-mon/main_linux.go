// https://github.com/torvalds/linux/blob/master/tools/gpio/gpio-event-mon.c
//
package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"syscall"

	"github.com/mkch/gpio"
)

func main() {
	flag.Usage = func() {
		fmt.Fprintln(flag.CommandLine.Output(), `Usage: gpio-event-mon [options]...
Listen to events on GPIO lines, 0->1 1->0`)
		flag.PrintDefaults()
		fmt.Fprintln(flag.CommandLine.Output(), `
Example:
gpio-event-mon -n gpiochip0 -o 4 -r -f`)
	}
	deviceName := flag.String("n", "", "Listen on GPIOs on a `name`d device (must be stated)")
	line := flag.Int("o", 0, "The `offset` to monitor")
	openDrain := flag.Bool("d", false, "Set line as open drain")
	openSource := flag.Bool("s", false, "Set line as open source")
	risingEdge := flag.Bool("r", false, "Listen for rising edges")
	fallingEdge := flag.Bool("f", false, "Listen for rising edges")
	loops := flag.Uint("c", 0, "Do <`n`> loops (optional, infinite loop if not stated)")
	flag.Parse()

	var handleFlags = gpio.Input
	var eventFlags gpio.EventFlag

	if *openDrain {
		handleFlags |= gpio.OpenDrain
	}
	if *openSource {
		handleFlags |= gpio.OpenSource
	}
	if *risingEdge {
		eventFlags |= gpio.RisingEdge
	}
	if *fallingEdge {
		eventFlags |= gpio.FallingEdge
	}

	if len(*deviceName) == 0 || *line == -1 {
		flag.Usage()
		os.Exit(-1)
	}

	if eventFlags == 0 {
		fmt.Println("No flags specified, listening on both rising and falling edges")
		eventFlags = gpio.BothEdges
	}

	err := monitorDevice(*deviceName, *line, handleFlags, eventFlags, *loops)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		var errno syscall.Errno
		if errors.As(err, &errno) {
			os.Exit(-int(errno))
		}
		os.Exit(-1)
	}
}

func monitorDevice(deviceName string, lineOffset int, handleFlags gpio.LineFlag, eventFlags gpio.EventFlag, loops uint) (err error) {
	chip, err := gpio.OpenChip(deviceName)
	if err != nil {
		return
	}
	defer chip.Close()

	line, err := chip.OpenLineWithEvents(uint32(lineOffset), handleFlags, eventFlags, "gpio-event-mon")
	if err != nil {
		return
	}
	defer line.Close()

	// Read initial states
	value, err := line.Value()
	if err != nil {
		return
	}

	fmt.Printf("Monitoring line %v on %v\n", lineOffset, deviceName)
	fmt.Printf("Initial line value: %v\n", value)

	var i uint
	for t := range line.Events() {
		fmt.Printf("GPIO EVENT %v: ", t)
		value, err = line.Value()
		if err != nil {
			return
		}
		if value == 0 {
			fmt.Println("falling edge")
		} else {
			fmt.Println("rising edge")
		}
		i++
		if i == loops {
			break
		}
	}
	return
}
