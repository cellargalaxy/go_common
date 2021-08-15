package util

import (
	"sync"
	"time"
)

var rateLimitMap sync.Map
var rateTsMap sync.Map

func AddRateLimit(key string, limit time.Duration) {
	rateLimitMap.Store(key, limit)
}

func RateLimit(key string) bool {
	limitP, ok := rateLimitMap.Load(key)
	if limitP == nil || !ok {
		return false
	}
	now := time.Now().Unix()
	tsP, ok := rateTsMap.Load(key)
	if tsP == nil || !ok {
		rateTsMap.Store(key, now)
		return false
	}
	limit := limitP.(time.Duration)
	ts := tsP.(int64)
	before := time.Unix(ts, 0)
	if now < before.Add(limit).Unix() {
		return true
	}
	rateTsMap.Store(key, now)
	return false
}
