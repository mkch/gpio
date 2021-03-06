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

	go func() {
		var v = time.Now().Unix()
		for i := 0; i < 10; i++ {
			_, err := unix.Write(pipe[1], (*[unsafe.Sizeof(v)]byte)(unsafe.Pointer(&v))[:])
			t.AssertNoError(err)
			time.Sleep(time.Microsecond * time.Duration(rand.Int63n(500)))
			v++
		}
		// Exit notification.
		v = 0
		_, err := unix.Write(pipe[1], (*[unsafe.Sizeof(v)]byte)(unsafe.Pointer(&v))[:])
		t.AssertNoError(err)
	}()

	events, err := fdevents.New(pipe[0], false, unix.EPOLLIN, func(fd int) *fdevents.Event {
		var v int64
		_, err := io.ReadFull(sys.FdReader(fd), (*[unsafe.Sizeof(v)]byte)(unsafe.Pointer(&v))[:])
		t.AssertNoError(err)
		return &fdevents.Event{Time: time.Unix(v, 0)}
	})
	t.AssertNoError(err)

	defer func() { t.Assert(events.Close(), NotEquals(nil)) }()

	var received []int64

for_loop:
	for {
		select {
		case v, ok := <-events.Events():
			if !ok {
				break for_loop
			}
			if v.Time.Unix() == 0 {
				t.AssertNoError(events.Close())
			}
			received = append(received, v.Time.Unix())

		}
	}

	t.AssertTrue(len(received) > 0)
	t.Assert(received, Matches(func(v interface{}) bool {
		return received[len(received)-1] == 0
	}).SetMessage(fmt.Sprintf("The last element of <%v> is expected to be <%v>", received, 0)))

}

func TestFdEventsClose(t1 *testing.T) {
	t := NewTB(t1)

	var pipe [2]int
	t.AssertNoError(unix.Pipe(pipe[:]))
	// pipe[0] will be closed by events.
	defer unix.Close(pipe[1])

	go func() {
		var v int64
		_, err := unix.Write(pipe[1], (*[unsafe.Sizeof(v)]byte)(unsafe.Pointer(&v))[:])
		t.AssertNoError(err)
	}()

	events, err := fdevents.New(pipe[0], true /*close fd on close*/, unix.EPOLLIN, func(fd int) *fdevents.Event {
		var v int64
		_, err := io.ReadFull(sys.FdReader(fd), (*[unsafe.Sizeof(v)]byte)(unsafe.Pointer(&v))[:])
		t.AssertNoError(err)
		return &fdevents.Event{Time: time.Unix(v, 0)}
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
