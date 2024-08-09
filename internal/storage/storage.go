package storage

import "github.com/pkg/errors"

var (
	ErrMetricMalformed = errors.New("metric malformed")
	ErrNotSupported    = errors.New("not supported")
)
