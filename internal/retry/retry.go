// Inprised by https://github.com/matryer/try
package retry

import (
	"errors"
	"time"
)

var (
	ErrMaxRetryExeeded = errors.New("maximum retry exided")
	delays             = []time.Duration{time.Second * 1, time.Second * 3, time.Second * 5}
)

const (
	MaximummAttempts = 3
)

type Func func(attempt int) (retry bool, err error)

func Do(fn Func) error {
	var err error
	var retry bool

	attempt := 1
	for {
		retry, err = fn(attempt)
		time.Sleep(delays[attempt-1])

		if !retry || err == nil {
			break
		}
		attempt++
		if attempt > MaximummAttempts {
			return ErrMaxRetryExeeded
		}
	}

	return err
}
