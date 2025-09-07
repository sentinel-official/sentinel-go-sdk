package utils

import (
	"math/rand/v2"
)

func RandomPort() uint16 {
	n := 1<<16 - 1<<10

	return uint16(rand.IntN(n) + 1<<10)
}
