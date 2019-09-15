package gpio_test

import (
	"flag"
	"os"
	"path/filepath"
	"testing"

	. "github.com/mkch/asserting"
	"github.com/mkch/gpio"
)

// The GPIO chip device to use in tests.
var chipDev string
var inputLine uint
var outputLine uint

func TestMain(m *testing.M) {
	flag.StringVar(&chipDev, "chip", "/dev/gpiochip0", "GPIO chip device used in tests, e.g. /dev/gpiochip0")
	flag.UintVar(&inputLine, "in-line", 2, "The offset of input GPIO line on chip used in tests")
	flag.UintVar(&outputLine, "out-line", 2, "The offset of output GPIO line on chip used in tests")
	flag.Parse()

	os.Exit(m.Run())
}

func TestOpenChip(t1 *testing.T) {
	t := NewTB(t1)
	chip, err := gpio.OpenChip(chipDev)
	t.Assert(ValueErrorFatal(chip, err), NotEquals(nil).SetFatal())
	t.AssertNoError(chip.Close())
}

func TestChipInfo(t1 *testing.T) {
	t := NewTB(t1)
	chip, err := gpio.OpenChip(chipDev)
	t.Assert(ValueErrorFatal(chip, err), NotEquals(nil).SetFatal())
	defer func() { t.AssertNoError(chip.Close()) }()

	info, err := chip.Info()
	t.Assert(ValueError(info, err), Matches(func(v interface{}) bool {
		return v.(gpio.ChipInfo).Name == filepath.Base(chipDev)
	}))
}

func TestLineInfo(t1 *testing.T) {
	t := NewTB(t1)
	chip, err := gpio.OpenChip(chipDev)
	t.Assert(ValueErrorFatal(chip, err), NotEquals(nil).SetFatal())
	defer func() { t.AssertNoError(chip.Close()) }()

	t.Assert(ValueError(chip.LineInfo(uint32(inputLine))), NotEquals(nil).SetFatal())
}

func TestOpenInputLine(t1 *testing.T) {
	t := NewTB(t1)
	chip, err := gpio.OpenChip(chipDev)
	t.Assert(ValueErrorFatal(chip, err), NotEquals(nil).SetFatal())
	defer func() { t.AssertNoError(chip.Close()) }()

	const consumer = "123456789abcdefghijklmnopqrstuv"
	t.Assert(len(consumer), Equals(31).SetFatal())

	type args struct {
		flags    gpio.LineFlag
		consumer string
	}
	type wantInfo struct {
		consumer  string
		activeLow bool
	}
	type testCase struct {
		name     string
		args     args
		wantInfo wantInfo
	}
	tests := []testCase{
		testCase{
			name: "consumer",
			args: args{
				flags:    gpio.Input,
				consumer: consumer,
			},
			wantInfo: wantInfo{
				consumer: consumer,
			},
		},
		testCase{
			name: "active-low",
			args: args{
				flags:    gpio.Input | gpio.ActiveLow,
				consumer: consumer,
			},
			wantInfo: wantInfo{
				consumer:  consumer,
				activeLow: true,
			},
		},
	}
	for _, tt := range tests {
		t1.Run(tt.name, func(t1 *testing.T) {
			t := NewTB(t1)
			line, err := chip.OpenLine(uint32(inputLine), 0, tt.args.flags, tt.args.consumer)
			t.Assert(ValueError(line, err), NotEquals(nil).SetFatal())

			gotInfo, err := chip.LineInfo(uint32(inputLine))
			t.AssertNoError(err)
			t.AssertEqual(gotInfo.Output(), false)
			t.AssertEqual(gotInfo.Consumer, tt.wantInfo.consumer)
			t.AssertEqual(gotInfo.ActiveLow(), tt.wantInfo.activeLow)
			t.AssertEqual(gotInfo.OpenDrain(), false)
			t.AssertEqual(gotInfo.OpenSource(), false)

			t.AssertNoError(line.Close())
		})
	}
}

func TestOpenOutputLine(t1 *testing.T) {
	t := NewTB(t1)
	chip, err := gpio.OpenChip(chipDev)
	t.Assert(ValueErrorFatal(chip, err), NotEquals(nil).SetFatal())
	defer func() { t.AssertNoError(chip.Close()) }()

	const consumer = "a"

	type args struct {
		flags    gpio.LineFlag
		consumer string
	}
	type wantInfo struct {
		consumer   string
		activeLow  bool
		openDrain  bool
		openSource bool
	}
	type testCase struct {
		name     string
		args     args
		wantInfo wantInfo
	}
	tests := []testCase{
		testCase{
			name: "consumer",
			args: args{
				flags:    gpio.Output,
				consumer: consumer,
			},
			wantInfo: wantInfo{
				consumer: consumer,
			},
		},
		testCase{
			name: "active-low",
			args: args{
				consumer: consumer,
				flags:    gpio.Output | gpio.ActiveLow,
			},
			wantInfo: wantInfo{
				consumer:  consumer,
				activeLow: true,
			},
		},
		testCase{
			name: "empty-consumer",
			args: args{
				flags: gpio.Output,
			},
			wantInfo: wantInfo{
				// The kernel return ? if the line is occupied but without a consumer abel.
				// This behavior is not documented.
				consumer: "?",
			},
		},
	}
	for _, tt := range tests {
		t1.Run(tt.name, func(t1 *testing.T) {
			t := NewTB(t1)
			line, err := chip.OpenLine(uint32(inputLine), 0, tt.args.flags, tt.args.consumer)
			t.Assert(ValueError(line, err), NotEquals(nil).SetFatal())
			t.AssertNoError(line.SetValue(1))

			gotInfo, err := chip.LineInfo(uint32(inputLine))
			t.AssertNoError(err)
			t.AssertEqual(gotInfo.Output(), true)
			t.AssertEqual(gotInfo.Consumer, tt.wantInfo.consumer)
			t.AssertEqual(gotInfo.ActiveLow(), tt.wantInfo.activeLow)
			t.AssertEqual(gotInfo.OpenDrain(), tt.wantInfo.openDrain)
			t.AssertEqual(gotInfo.OpenSource(), tt.wantInfo.openSource)

			t.AssertNoError(line.Close())
		})
	}
}
