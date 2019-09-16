package fdevents_test

import (
	"fmt"
	"io"
	"math/rand"
	"testing"
	"time"
	"unsafe"

	. "github.com/mkch/asserting"
	"github.com/mkch/gpio/internal/fdevents"
	"github.com/mkch/gpio/internal/sys"
	"golang.org/x/sys/unix"
)

func TestFdEvents(t1 *testing.T) {
	t := NewTB(t1)

	var pipe [2]int
	t.AssertNoError(unix.Pipe(pipe[:]))
	defer unix.Close(pipe[0])
	defer unix.Close(pipe[1])

	var lastSend int64

	var done = make(chan struct{})

	go func() {
		defer close(done)
		var v = time.Now().Unix()
		for i := 0; i < 10; i++ {
			_, err := unix.Write(pipe[1], (*[unsafe.Sizeof(v)]byte)(unsafe.Pointer(&v))[:])
			t.AssertNoError(err)
			lastSend = v
			time.Sleep(time.Microsecond * time.Duration(rand.Int63n(100)))
			v++
		}
	}()

	events, err := fdevents.New(pipe[0], unix.EPOLLIN, func(fd int) time.Time {
		var v int64
		_, err := io.ReadFull(sys.FdReader(fd), (*[unsafe.Sizeof(v)]byte)(unsafe.Pointer(&v))[:])
		t.AssertNoError(err)
		return time.Unix(v, 0)
	})
	t.AssertNoError(err)

	defer func() { t.AssertNoError(events.Close()) }()

	var received []int64

for_loop:
	for {
		select {
		case v, ok := <-events.Events():
			if !ok {
				break for_loop
			}
			received = append(received, v.Unix())
		case <-done:
			break for_loop
		}
	}

	t.AssertTrue(len(received) > 0)
	t.Assert(received, Matches(func(v interface{}) bool {
		return received[len(received)-1] == lastSend
	}).SetMessage(fmt.Sprintf("The last element of <%v> is expected to be <%v>", received, lastSend)))

}

func TestFdEventsClose(t1 *testing.T) {
	t := NewTB(t1)

	var pipe [2]int
	t.AssertNoError(unix.Pipe(pipe[:]))
	defer unix.Close(pipe[0])
	defer unix.Close(pipe[1])

	go func() {
		var v int64
		_, err := unix.Write(pipe[1], (*[unsafe.Sizeof(v)]byte)(unsafe.Pointer(&v))[:])
		t.AssertNoError(err)
	}()

	events, err := fdevents.New(pipe[0], unix.EPOLLIN, func(fd int) time.Time {
		var v int64
		_, err := io.ReadFull(Fd(fd), (*[unsafe.Sizeof(v)]byte)(unsafe.Pointer(&v))[:])
		t.AssertNoError(err)
		return time.Unix(v, 0)
	})
	t.AssertNoError(err)

	defer func() { t.AssertMatch(events.Close(), func(v interface{}) bool { return v.(error) != nil }) }()

for_loop:
	for {
		select {
		case _, ok := <-events.Events():
			if !ok {
				break for_loop
			}
			t.AssertNoError(events.Close())
		}
	}

}
