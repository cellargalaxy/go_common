package util

import (
	"testing"
	"time"
)

func TestLocalCache(t *testing.T) {
	ctx := GenCtx()

	var cache1 = NewLocalCache[string]()
	cache2 := cache1

	cache1.Set(ctx, "aaa", "aaa", time.Hour)
	value, ok := cache2.Get(ctx, "aaa")
	if value != "aaa" || !ok {
		t.Errorf("value != \"aaa\" || !ok")
		return
	}

	cache1.GetWithTimeout(ctx, "bbb", time.Hour, func() (string, error) {
		return "bbb", nil
	})
	value, err := cache2.GetWithTimeout(ctx, "bbb", time.Hour, func() (string, error) {
		return "ccc", nil
	})
	if value != "bbb" || err != nil {
		t.Errorf("value != \"bbb\" || !ok: %+v", err)
		return
	}

	if !cache1.lock.TryLock() {
		t.Errorf("if !cache1.lock.TryLock() {")
		return
	}
	if cache2.lock.TryLock() {
		t.Errorf("if cache2.lock.TryLock() {")
		return
	}
}
