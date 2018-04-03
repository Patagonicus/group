package group

import (
	"context"
)

// Actor represents a long running task that can be interrupted.
type Actor interface {
	// Execute runs some computation.
	Execute() error
	// Interrupt stops the actor. Interrupt should not wait for Execute to
	// return and it must be safe to be called multiple times.
	Interrupt()
}

type actor struct {
	execute   func() error
	interrupt func()
}

// New allows easy creation of an actor from two functions.
func New(execute func() error, interrupt func()) Actor {
	return actor{
		execute:   execute,
		interrupt: interrupt,
	}
}

func (a actor) Execute() error {
	return a.execute()
}

func (a actor) Interrupt() {
	a.interrupt()
}

// Done returns an actor that immediately returs the given result when
// executed. This can be used when there is a problem setting up an
// actor to get the error to the function calling Run.
func Done(result error) Actor {
	return New(
		func() error {
			return result
		},
		func() {},
	)
}

// WithContext creates an actor from a context and a function that takes a
// context.
func WithContext(ctx context.Context, execute func(context.Context) error) Actor {
	c, cancel := context.WithCancel(ctx)
	return New(
		func() error {
			return execute(c)
		},
		cancel,
	)
}

// WithChannel creates an actor from a function that takes a channel. When the
// actor is interrupted the channel given to the function is closed.
func WithChannel(execute func(<-chan struct{}) error) Actor {
	c := make(chan struct{})
	return New(
		func() error {
			return execute(c)
		},
		func() {
			close(c)
		},
	)
}

// Run starts all actors and waits for the first one to finish. Once one actor
// has returned the others are interrupted. Run then waits for all actors to
// finish.
//
// Run returns the error of the first actor to finish. If that actor returns
// nil, then Run will also return nil, even if other actors return a non-nil
// error.
//
// If Run is called without any actors it will return nil immediately.
func Run(actors ...Actor) error {
	if len(actors) == 0 {
		return nil
	}

	errors := make(chan error, len(actors))
	for _, a := range actors {
		go func(a Actor) {
			errors <- a.Execute()
		}(a)
	}

	err := <-errors

	for _, a := range actors {
		a.Interrupt()
	}

	for i := 1; i < len(actors); i++ {
		<-errors
	}

	return err
}
