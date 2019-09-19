package main

import (
	"flag"
	"log"
	"os"
	"os/signal"

	"github.com/mkch/gpio"
)

func main() {
	var btnOffset, ledOffset uint
	flag.UintVar(&btnOffset, "btn", 2, "line `offset` of button")
	flag.UintVar(&ledOffset, "led", 17, "line `offset` of LED")
	flag.Parse()
	devices := gpio.ChipDevices()
	if len(devices) == 0 {
		log.Fatal("No GPIO chip")
	}
	chip := mustChip(gpio.OpenChip(devices[0]))
	defer chip.Close()

	led := mustLine(chip.OpenLine(uint32(ledOffset), 0, gpio.Output, "led"))
	defer led.Close()

	btn := mustLineEvt(chip.OpenLineWithEvents(uint32(btnOffset), gpio.Input, gpio.BothEdges, "btn"))
	defer btn.Close()

	exit := make(chan os.Signal, 2)
	signal.Notify(exit, os.Interrupt)

	for {
		select {
		case event, ok := <-btn.Events():
			if !ok {
				return
			}
			if event.RisingEdge {
				must(led.SetValue(0))
			} else {
				must(led.SetValue(1))
			}
		case <-exit:
			return
		}
	}
}

func must(err error) {
	if err != nil {
		log.Panic(err)
	}
}

func mustValue(v byte, err error) byte {
	if err != nil {
		log.Panic(err)
	}
	return v
}

func mustChip(chip *gpio.Chip, err error) *gpio.Chip {
	if err != nil {
		log.Panic(err)
	}
	return chip
}

func mustLine(line *gpio.Line, err error) *gpio.Line {
	if err != nil {
		log.Panic(err)
	}
	return line
}

func mustLineEvt(line *gpio.LineWithEvent, err error) *gpio.LineWithEvent {
	if err != nil {
		log.Panic(err)
	}
	return line
}
