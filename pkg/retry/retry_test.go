package retry_test

import (
	"fmt"

	"github.com/nbvehbq/go-metrics-harvester/pkg/retry"
)

func Example() {
	gen := func() chan error {
		c := make(chan error)
		go func() {
			c <- fmt.Errorf("error in operation")
			c <- nil
		}()
		return c
	}

	op := func() error {
		return <-gen()
	}

	retry.Do(op)
}
