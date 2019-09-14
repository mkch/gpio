package sys

import (
	"fmt"
	"reflect"
	"testing"

	. "github.com/mkch/asserting"
	"github.com/mkch/gpio/internal/c"
)

func TestConsts(t *testing.T) {
	type testCase struct {
		name string
		arg  uint32
		want uint32
	}
	tests := []testCase{
		testCase{
			"GPIO_GET_CHIPINFO_IOCTL",
			GPIO_GET_CHIPINFO_IOCTL,
			c.GPIO_GET_CHIPINFO_IOCTL,
		},
		testCase{
			"GPIO_GET_LINEINFO_IOCTL",
			GPIO_GET_LINEINFO_IOCTL,
			c.GPIO_GET_LINEINFO_IOCTL,
		},
		testCase{
			"GPIO_GET_LINEHANDLE_IOCTL",
			GPIO_GET_LINEHANDLE_IOCTL,
			c.GPIO_GET_LINEHANDLE_IOCTL,
		},
		testCase{
			"GPIOHANDLE_SET_LINE_VALUES_IOCTL",
			GPIOHANDLE_SET_LINE_VALUES_IOCTL,
			c.GPIOHANDLE_SET_LINE_VALUES_IOCTL,
		},
		testCase{
			"GPIOHANDLE_GET_LINE_VALUES_IOCTL",
			GPIOHANDLE_GET_LINE_VALUES_IOCTL,
			c.GPIOHANDLE_GET_LINE_VALUES_IOCTL,
		},
		testCase{
			"GPIO_GET_LINEEVENT_IOCTL",
			GPIO_GET_LINEEVENT_IOCTL,
			c.GPIO_GET_LINEEVENT_IOCTL,
		},
		testCase{
			"GPIOLINE_FLAG_KERNEL",
			GPIOLINE_FLAG_KERNEL,
			c.GPIOLINE_FLAG_KERNEL,
		},
		testCase{
			"GPIOLINE_FLAG_IS_OUT",
			GPIOLINE_FLAG_IS_OUT,
			c.GPIOLINE_FLAG_IS_OUT,
		},
		testCase{
			"GPIOLINE_FLAG_ACTIVE_LOW",
			GPIOLINE_FLAG_ACTIVE_LOW,
			c.GPIOLINE_FLAG_ACTIVE_LOW,
		},
		testCase{
			"GPIOLINE_FLAG_OPEN_DRAIN",
			GPIOLINE_FLAG_OPEN_DRAIN,
			c.GPIOLINE_FLAG_OPEN_DRAIN,
		},
		testCase{
			"GPIOLINE_FLAG_OPEN_SOURCE",
			GPIOLINE_FLAG_OPEN_SOURCE,
			c.GPIOLINE_FLAG_OPEN_SOURCE,
		},
		testCase{
			"GPIOHANDLE_REQUEST_INPUT",
			GPIOHANDLE_REQUEST_INPUT,
			c.GPIOHANDLE_REQUEST_INPUT,
		},
		testCase{
			"GPIOHANDLE_REQUEST_OUTPUT",
			GPIOHANDLE_REQUEST_OUTPUT,
			c.GPIOHANDLE_REQUEST_OUTPUT,
		},
		testCase{
			"GPIOHANDLE_REQUEST_ACTIVE_LOW",
			GPIOHANDLE_REQUEST_ACTIVE_LOW,
			c.GPIOHANDLE_REQUEST_ACTIVE_LOW,
		},
		testCase{
			"GPIOHANDLE_REQUEST_OPEN_DRAIN",
			GPIOHANDLE_REQUEST_OPEN_DRAIN,
			c.GPIOHANDLE_REQUEST_OPEN_DRAIN,
		},
		testCase{
			"GPIOHANDLE_REQUEST_OPEN_SOURCE",
			GPIOHANDLE_REQUEST_OPEN_SOURCE,
			c.GPIOHANDLE_REQUEST_OPEN_SOURCE,
		},
		testCase{
			"GPIOEVENT_REQUEST_RISING_EDGE",
			GPIOEVENT_REQUEST_RISING_EDGE,
			c.GPIOEVENT_REQUEST_RISING_EDGE,
		},
		testCase{
			"GPIOEVENT_REQUEST_FALLING_EDGE",
			GPIOEVENT_REQUEST_FALLING_EDGE,
			c.GPIOEVENT_REQUEST_FALLING_EDGE,
		},
		testCase{
			"GPIOEVENT_REQUEST_BOTH_EDGES",
			GPIOEVENT_REQUEST_BOTH_EDGES,
			c.GPIOEVENT_REQUEST_BOTH_EDGES,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t1 *testing.T) {
			t := NewTB(t1)
			t.AssertEqual(tt.arg, tt.want)
		})
	}
}

func cmpType(t TB, t1, t2 reflect.Type) {
	t.Helper()
	if t1 == t2 {
		return
	}
	t.Assert(t1.Size(), Equals(t2.Size()).SetMessageFunc(func() string {
		return fmt.Sprintf("sizeof(%v)=%v, but sizeof(%v)=%v", t1, t1.Size(), t2, t2.Size())
	}))
	switch t1.Kind() {
	case reflect.Struct:
		t.Assert(t1.Kind(), Equals(t2.Kind()).SetMessageFunc(func() string {
			return fmt.Sprintf("<%v> != <%v>", t1.Kind(), t2.Kind())
		}))
		t.Assert(t1.NumField(), Equals(t2.NumField()).SetMessageFunc(func() string {
			return fmt.Sprintf("<%v> has <%v> fields, but <%v> has <%v> fields", t1, t1.NumField(), t2, t2.NumField())
		}))
		numField := t1.NumField()
		for i := 0; i < numField; i++ {
			tf1 := t1.Field(i)
			tf2 := t2.Field(i)
			t.Assert(tf1.Offset, Equals(tf2.Offset).SetMessageFunc(func() string {
				return fmt.Sprintf("offset of <%v>.<%v> is %v, but <%v>.<%v> is %v", t1, tf1.Name, tf1.Offset, t2, tf2.Name, tf2.Offset)
			}))
			cmpType(t, tf1.Type, tf2.Type)
		}
	case reflect.Array:
		t.Assert(t1.Kind(), Equals(t2.Kind()).SetMessageFunc(func() string {
			return fmt.Sprintf("<%v> != <%v>", t1, t2)
		}))
		cmpType(t, t1.Elem(), t2.Elem())
		t.Assert(t1.Len(), Equals(t2.Len()).SetMessageFunc(func() string {
			return fmt.Sprintf("len(%v) != len(%v)", t1, t2)
		}))
	default:
		t.Assert(t1.ConvertibleTo(t2), Equals(true).SetMessageFunc(func() string {
			return fmt.Sprintf("%v is not convertable to %v", t1, t2)
		}))
	}
}

func TestStruct(t1 *testing.T) {
	t := NewTB(t1)
	cmpType(t, reflect.TypeOf(GPIOChipInfo{}), reflect.TypeOf(c.GPIOChipInfo{}))
	cmpType(t, reflect.TypeOf(GPIOLineInfo{}), reflect.TypeOf(c.GPIOLineInfo{}))
	cmpType(t, reflect.TypeOf(GPIOHandleRequest{}), reflect.TypeOf(c.GPIOHandleRequest{}))
	cmpType(t, reflect.TypeOf(GPIOEventRequest{}), reflect.TypeOf(c.GPIOEventRequest{}))
	cmpType(t, reflect.TypeOf(GPIOEventData{}), reflect.TypeOf(c.GPIOEventData{}))
}
