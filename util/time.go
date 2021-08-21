package util

import (
	"math/rand"
	"time"
)

func WareDuration(duration time.Duration) time.Duration {
	rate := 1 - (rand.Float64() * rand.Float64())
	rate = 0.9 + (0.2 * rate)
	ns := int64(duration)
	ns = int64(float64(ns) * rate)
	return time.Duration(ns)
}
