// Inprised by https://github.com/matryer/try
package retry

import (
	"time"
)

var (
	delays = []time.Duration{time.Second * 1, time.Second * 3, time.Second * 5}
)

type Func func() (err error)

func Do(fn Func) (err error) {
	for _, delay := range delays {
		if err = fn(); err == nil {
			break
		}
		time.Sleep(delay)
	}

	return err
}
