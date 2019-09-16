package main

import (
	"flag"
	"log"
	"os"
	"os/signal"

	gpio "github.com/mkch/gpio/gpiosysfs"
)

func main() {
	var btnOffset, ledOffset int
	flag.IntVar(&btnOffset, "btn", 2, "line `offset` of button")
	flag.IntVar(&ledOffset, "led", 17, "line `offset` of LED")
	flag.Parse()

	led := mustPin(gpio.OpenPin(ledOffset))
	defer led.Close()

	must(led.SetDirection(gpio.Out))

	btn := mustPinEvt(gpio.OpenPinWithEvents(btnOffset))
	btn.SetEdge(gpio.Both)

	defer btn.Close()

	exit := make(chan os.Signal, 2)
	signal.Notify(exit, os.Interrupt)

	for {
		select {
		case _, ok := <-btn.Events():
			if !ok {
				return
			}
			if mustValue(btn.Value()) == 1 {
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

func mustPin(line *gpio.Pin, err error) *gpio.Pin {
	if err != nil {
		log.Panic(err)
	}
	return line
}

func mustPinEvt(line *gpio.PinWithEvent, err error) *gpio.PinWithEvent {
	if err != nil {
		log.Panic(err)
	}
	return line
}
