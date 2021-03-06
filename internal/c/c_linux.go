// Package c is used is test code only.
package c

/*
#include <linux/gpio.h>
#include <string.h>
*/
import "C"

var (
	GPIO_GET_CHIPINFO_IOCTL          = uint32(C.GPIO_GET_CHIPINFO_IOCTL)
	GPIO_GET_LINEINFO_IOCTL          = uint32(C.GPIO_GET_LINEINFO_IOCTL)
	GPIO_GET_LINEHANDLE_IOCTL        = uint32(C.GPIO_GET_LINEHANDLE_IOCTL)
	GPIOHANDLE_SET_LINE_VALUES_IOCTL = uint32(C.GPIOHANDLE_SET_LINE_VALUES_IOCTL)
	GPIOHANDLE_GET_LINE_VALUES_IOCTL = uint32(C.GPIOHANDLE_GET_LINE_VALUES_IOCTL)
	GPIO_GET_LINEEVENT_IOCTL         = uint32(C.GPIO_GET_LINEEVENT_IOCTL)
)

var (
	GPIOLINE_FLAG_KERNEL      = uint32(C.GPIOLINE_FLAG_KERNEL)
	GPIOLINE_FLAG_IS_OUT      = uint32(C.GPIOLINE_FLAG_IS_OUT)
	GPIOLINE_FLAG_ACTIVE_LOW  = uint32(C.GPIOLINE_FLAG_ACTIVE_LOW)
	GPIOLINE_FLAG_OPEN_DRAIN  = uint32(C.GPIOLINE_FLAG_OPEN_DRAIN)
	GPIOLINE_FLAG_OPEN_SOURCE = uint32(C.GPIOLINE_FLAG_OPEN_SOURCE)
)

var (
	GPIOHANDLE_REQUEST_INPUT       = uint32(C.GPIOHANDLE_REQUEST_INPUT)
	GPIOHANDLE_REQUEST_OUTPUT      = uint32(C.GPIOHANDLE_REQUEST_OUTPUT)
	GPIOHANDLE_REQUEST_ACTIVE_LOW  = uint32(C.GPIOHANDLE_REQUEST_ACTIVE_LOW)
	GPIOHANDLE_REQUEST_OPEN_DRAIN  = uint32(C.GPIOHANDLE_REQUEST_OPEN_DRAIN)
	GPIOHANDLE_REQUEST_OPEN_SOURCE = uint32(C.GPIOHANDLE_REQUEST_OPEN_SOURCE)
)

var (
	GPIOEVENT_REQUEST_RISING_EDGE  = uint32(C.GPIOEVENT_REQUEST_RISING_EDGE)
	GPIOEVENT_REQUEST_FALLING_EDGE = uint32(C.GPIOEVENT_REQUEST_FALLING_EDGE)
	GPIOEVENT_REQUEST_BOTH_EDGES   = uint32(C.GPIOEVENT_REQUEST_BOTH_EDGES)
)

type GPIOChipInfo = C.struct_gpiochip_info
type GPIOLineInfo = C.struct_gpioline_info
type GPIOHandleRequest = C.struct_gpiohandle_request
type GPIOEventRequest = C.struct_gpioevent_request
type GPIOEventData = C.struct_gpioevent_data
