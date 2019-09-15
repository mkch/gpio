// https://github.com/torvalds/linux/blob/master/tools/gpio/lsgpio.c
package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"
	"syscall"

	"github.com/mkch/gpio"
)

func main() {
	flag.Usage = func() {
		fmt.Fprintln(flag.CommandLine.Output(),
			`Usage: lsgpio [options]
List GPIO chips, lines and states`)
		flag.PrintDefaults()
	}
	var deviceName = flag.String("n", "", "List GPIOs on a `name`d device")
	flag.Parse()

	var err error
	if len(*deviceName) > 0 {
		err = listDevice("/dev/" + *deviceName)
	} else {
		devices := gpio.ChipDevices()
		for _, dev := range devices {
			err = listDevice(dev)
			if err != nil {
				break
			}
		}
	}

	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		var errno syscall.Errno
		if errors.As(err, &errno) {
			os.Exit(-int(errno))
		}
		os.Exit(-1)
	}
}

func listDevice(deviceName string) (err error) {
	chip, err := gpio.OpenChip(deviceName)
	if err != nil {
		return
	}
	// Inspect this GPIO chip
	chipInfo, err := chip.Info()
	if err != nil {
		return
	}
	fmt.Printf("GPIO chip: %v, \"%v\", %v GPIO lines\n", chipInfo.Name, chipInfo.Label, chipInfo.NumLines)
	// Loop over the lines and print info
	for i := uint32(0); i < chipInfo.NumLines; i++ {
		var lineInfo gpio.LineInfo
		lineInfo, err = chip.LineInfo(i)
		if err != nil {
			return
		}
		fmt.Printf("\tline %2d:", lineInfo.Offset)
		if len(lineInfo.Name) > 0 {
			fmt.Printf(` "%v"`, lineInfo.Name)
		} else {
			fmt.Print(" unnamed")
		}
		if len(lineInfo.Consumer) > 0 {
			fmt.Printf(` "%v"`, lineInfo.Consumer)
		} else {
			fmt.Print(" unused")
		}
		var flags []string
		if lineInfo.Kernel() {
			flags = append(flags, "kernel")
		}
		if lineInfo.Output() {
			flags = append(flags, "output")
		}
		if lineInfo.ActiveLow() {
			flags = append(flags, "active-low")
		}
		if lineInfo.OpenDrain() {
			flags = append(flags, "open-drain")
		}
		if lineInfo.OpenSource() {
			flags = append(flags, "open-source")
		}
		fmt.Printf(" [%v]\n", strings.Join(flags, " "))
	}
	return
}
