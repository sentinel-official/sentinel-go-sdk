package utils

import (
	"time"
)

// UIntFromFloat64 converts a float64 to a uint.
func UIntFromFloat64(v float64) uint {
	return uint(v)
}

// UIntSecondsFromDuration converts a time.Duration to a uint representing seconds.
func UIntSecondsFromDuration(v time.Duration) uint {
	return UIntFromFloat64(v.Seconds())
}
