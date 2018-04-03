package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/Patagonicus/group"
)

func main() {
	ids := make(chan int)

	err := group.Run(
		createGeneratorActor(ids),
		createServerActor(ids),
		createInterruptActor(),
	)
	if err != nil {
		log.Fatal(err)
	}
}

func createGeneratorActor(ids chan<- int) group.Actor {
	return group.WithChannel(func(done <-chan struct{}) error {
		id := 0
		for {
			select {
			case ids <- id:
				id++
			case <-done:
				log.Printf("stopping generator")
				return nil
			}
		}
	})
}

func createInterruptActor() group.Actor {
	c := make(chan os.Signal, 1)
	signal.Notify(c)

	return group.WithChannel(func(done <-chan struct{}) error {
		select {
		case <-c:
			log.Printf("caught interrupt, shutting down")
			return errors.New("interrupted")
		case <-done:
			return nil
		}
	})
}

func createServerActor(ids <-chan int) group.Actor {

	server := http.Server{
		Addr:    ":8080",
		Handler: createIDHandler(ids),
	}

	return group.New(
		func() error {
			log.Printf("listening on :8080")
			return server.ListenAndServe()
		},
		func() {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			err := server.Shutdown(ctx)
			if err != nil {
				log.Printf("error shutting down HTTP server: %v", err)
			} else {
				log.Printf("webserver shut down successfully")
			}
		},
	)
}

func createIDHandler(ids <-chan int) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}

		// There is a race condition here. If a new connection is made after the
		// generator has stopped, the next line will block. We could do a select
		// with a timeout, but as this is only an example, we don't really care.
		// The http server will handle this.
		fmt.Fprintf(w, "%d\n", <-ids)
	})
}
