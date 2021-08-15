package util

import (
	"sync"
	"time"
)

var rateLimitMap sync.Map
var rateTsMap sync.Map

func AddRateLimit(key string, limit int64) {
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
	limit := limitP.(int64)
	ts := tsP.(int64)
	if now < ts+limit {
		return true
	}
	rateTsMap.Store(key, now)
	return false
}
