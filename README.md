# group [![go-doc](https://godoc.org/github.com/Patagonicus/group?status.svg)](https://godoc.org/github.com/Patagonicus/group) [![Build Status](https://travis-ci.org/Patagonicus/group.svg?branch=master)](https://travis-ci.org/Patagonicus/group) [![Coverage Status](https://coveralls.io/repos/github/Patagonicus/group/badge.svg?branch=master)](https://coveralls.io/github/Patagonicus/group?branch=master) [![Go Report Card](https://goreportcard.com/badge/github.com/Patagonicus/group)](https://goreportcard.com/report/github.com/Patagonicus/group)

In server applications you often have multiple long running goroutines that depend on each other, such as storage, a business model and a web server. When one of them fails the others usually can't continue to work, so it would be better to stop the whole program to have a human investigate. With group this is easy to do:

```go
err := group.Run(
  createStorageActor(),
  createModelActor(),
  createServerActor(),
  createInterruptActor(),
)
if err != nil {
  log.Fatal(err)
}
```

You create actors for the different parts of your program. An actor describes how to run a task and how to interrupt it. You can create an actor with `group.New()`:

```go
func New(execute func() error, interrupt func()) Actor
```

When you pass actors to `group.Run()` each actor's `execute` will be run in its own goroutine. Once one actor returns, all other actors are interrupted. `group.Run()` then returns the error of the actor that caused the others to be interrupted.

## Helper methods

There are a couple of helper methods for creating actors. These are useful if you are using `context.Context` or a channel to interrupt your goroutines:

```go
func WithContext(ctx context.Context, execute func(context.Context) error) Actor
func WithChannel(execute func(<-chan struct{}) error) Actor
```

The interrupt methods are automatically added to the actors.

There is also `group.Done(error)` which returns an actor that immediately returs the given error. This can be useful if you have a problem while creating an actor and have no other way of signalling this error to the calling code.
