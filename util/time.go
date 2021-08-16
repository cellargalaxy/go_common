package util

import (
	"math/rand"
	"time"
)

func WareDuration(duration time.Duration) time.Duration {
	ns := int64(duration)
	if ns <= 1 {
		return duration
	}
	rate := rand.Float64()
	rate = 0.5 + (0.5 * (1 - (rate * rate)))
	ns = int64(float64(ns) * rate)
	return time.Duration(ns)
}
