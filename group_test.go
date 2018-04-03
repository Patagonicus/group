package group_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Patagonicus/group"
)

// If everything is going well the timeout should never be a problem, so the
// exact value is not important.
var timeout = 10 * time.Second
var expectedErr = errors.New("expected")

func TestNoActors(t *testing.T) {
	withTimeout(t, timeout, func() {
		err := group.Run()
		if err != nil {
			t.Fatalf("expected nil, but got %v", err)
		}
	})
}

func TestSingleActor(t *testing.T) {
	withTimeout(t, timeout, func() {
		result := group.Run(
			group.Done(expectedErr),
		)

		if result != expectedErr {
			t.Fatalf("expected %v, but got %v", expectedErr, result)
		}
	})
}

func TestTwoActors_First(t *testing.T) {
	withTimeout(t, timeout, func() {
		result := group.Run(
			group.Done(expectedErr),
			waiter(nil),
		)

		if result != expectedErr {
			t.Fatalf("expected %v, but got %v", expectedErr, result)
		}
	})
}

func TestTwoActors_Second(t *testing.T) {
	withTimeout(t, timeout, func() {
		result := group.Run(
			waiter(nil),
			group.Done(expectedErr),
		)

		if result != expectedErr {
			t.Fatalf("expected %v, but got %v", expectedErr, result)
		}
	})
}

func TestMultipleActorsInterrupted(t *testing.T) {
	withTimeout(t, timeout, func() {
		const dummies = 5
		var actors []group.Actor

		c := make(chan struct{})
		for i := 0; i < dummies; i++ {
			done := make(chan struct{})
			actors = append(actors, group.New(
				func() error {
					<-done
					return nil
				},
				func() {
					c <- struct{}{}
					close(done)
				},
			))
		}

		actors = append(actors, group.Done(nil))

		go group.Run(actors...) // nolint:errcheck

		for i := 0; i < dummies; i++ {
			<-c
		}
	})
}

func TestWithContext(t *testing.T) {
	withTimeout(t, timeout, func() {
		actor := group.WithContext(context.Background(), func(ctx context.Context) error {
			<-ctx.Done()
			return expectedErr
		})

		c := make(chan error)
		go func() {
			c <- actor.Execute()
		}()

		actor.Interrupt()

		result := <-c

		if result != expectedErr {
			t.Fatalf("expected %v, but got %v", expectedErr, result)
		}
	})
}

func TestWithChannel(t *testing.T) {
	withTimeout(t, timeout, func() {
		actor := group.WithChannel(func(c <-chan struct{}) error {
			<-c
			return expectedErr
		})

		c := make(chan error)
		go func() {
			c <- actor.Execute()
		}()

		actor.Interrupt()

		result := <-c
		if result != expectedErr {
			t.Fatalf("expected %v, but got %v", expectedErr, result)
		}
	})
}

func TestDoneNil(t *testing.T) {
	withTimeout(t, timeout, func() {
		actor := group.Done(nil)

		result := actor.Execute()

		if result != nil {
			t.Fatalf("expected nil, but got %v", result)
		}
	})
}

func TestDone(t *testing.T) {
	withTimeout(t, timeout, func() {
		actor := group.Done(expectedErr)

		result := actor.Execute()

		if result != expectedErr {
			t.Fatalf("expected %v, but got %v", expectedErr, result)
		}
	})
}

// waiter returns an actor that will block until it is interrupted. Then it
// will return result.
func waiter(result error) group.Actor {
	return group.WithChannel(func(c <-chan struct{}) error {
		<-c
		return result
	})
}

func TestWaiterNil(t *testing.T) {
	withTimeout(t, timeout, func() {
		actor := waiter(nil)

		c := make(chan error)
		go func() {
			c <- actor.Execute()
		}()

		time.Sleep(50 * time.Millisecond)

		select {
		case result := <-c:
			t.Fatalf("waiter did not wait for interrupt, returned %v", result)
		default:
		}

		actor.Interrupt()

		result := <-c
		if result != nil {
			t.Fatalf("expected nil, but got %v", result)
		}
	})
}

func TestWaiter(t *testing.T) {
	withTimeout(t, timeout, func() {
		actor := waiter(expectedErr)

		c := make(chan error)
		go func() {
			c <- actor.Execute()
		}()

		time.Sleep(50 * time.Millisecond)

		select {
		case result := <-c:
			t.Fatalf("waiter did not wait for interrupt, returned %v", result)
		default:
		}

		actor.Interrupt()

		result := <-c
		if result != expectedErr {
			t.Fatalf("expected %v, but got %v", expectedErr, result)
		}
	})
}

// withTimeout runs f, but marks the test as failed if f does not complete
// within the given timeout.
func withTimeout(t *testing.T, timeout time.Duration, f func()) {
	t.Helper()

	done := make(chan struct{})
	timer := time.NewTimer(timeout)

	go func() {
		f()
		close(done)
	}()

	select {
	case <-done:
		if !timer.Stop() {
			<-timer.C
		}
	case <-timer.C:
		t.Fatalf("timed out")
	}
}
