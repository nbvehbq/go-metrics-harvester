// Package retry provides retry functionality.
//
//	var value string
//	err := try.Do(func() error {
//	  var err error
//	  value, err = SomeFunction()
//	})
//	if err != nil {
//	  log.Fatalln("error:", err)
//	}
package retry

import (
	"time"
)

var (
	delays = []time.Duration{time.Second * 1, time.Second * 3, time.Second * 5}
)

// Func represents functions that can be retried.
type Func func() (err error)

// Do keeps trying the function
func Do(fn Func) (err error) {
	for _, delay := range delays {
		if err = fn(); err == nil {
			break
		}
		time.Sleep(delay)
	}

	return err
}
