package util

import (
	"context"
	"fmt"
	"github.com/patrickmn/go-cache"
	"strings"
	"sync"
	"time"
)

var localCache = NewLocalCache[string]()

func existReqId(ctx context.Context, reqId string, duration time.Duration) bool {
	key := fmt.Sprintf("reqId-%s", reqId)
	_, ok := localCache.Get(ctx, key)
	localCache.Set(ctx, key, "", duration)
	return ok
}
func getHttpBan(ctx context.Context, address string) bool {
	address = strings.Split(address, "#")[0]
	address = strings.Split(address, "?")[0]
	key := fmt.Sprintf("httpBan-%s", address)
	_, ok := localCache.Get(ctx, key)
	return ok
}
func setHttpBan(ctx context.Context, address string, duration time.Duration) {
	address = strings.Split(address, "#")[0]
	address = strings.Split(address, "?")[0]
	key := fmt.Sprintf("httpBan-%s", address)
	localCache.Set(ctx, key, "", duration)
}

func NewLocalCache[T any]() LocalCache[T] {
	return LocalCache[T]{lock: &sync.Mutex{}, cache: cache.New(time.Minute, time.Minute)}
}

type LocalCache[T any] struct {
	lock    *sync.Mutex
	cache   *cache.Cache
	timeMap map[string]time.Time
}

func (this *LocalCache[T]) Get(ctx context.Context, key string) (T, bool) {
	var value T
	object, ok := this.cache.Get(key)
	if !ok {
		return value, false
	}
	value, ok = object.(T)
	return value, ok
}
func (this *LocalCache[T]) Set(ctx context.Context, key string, object T, duration time.Duration) {
	this.cache.Set(key, object, duration)
}
func (this *LocalCache[T]) Del(ctx context.Context, key string) {
	this.cache.Delete(key)
}
func (this *LocalCache[T]) GetWithTimeout(ctx context.Context, key string, duration time.Duration, get func() (T, error)) (T, error) {
	this.lock.Lock()
	defer this.lock.Unlock()

	object, ok := this.Get(ctx, key)
	cacheTime := this.timeMap[key]
	if ok && time.Now().Sub(cacheTime) <= duration {
		return object, nil
	}

	object, err := get()
	if err != nil {
		return object, err
	}

	this.Set(ctx, key, object, DurationMax)
	this.timeMap[key] = time.Now()

	return object, nil
}

// true:拿到锁；false:拿不到锁
func (this *LocalCache[T]) TryLock(ctx context.Context, key string, duration time.Duration) bool {
	this.lock.Lock()
	defer this.lock.Unlock()

	_, ok := this.Get(ctx, key)
	if ok {
		return false
	}

	var object T
	this.Set(ctx, key, object, duration)
	return true
}
func (this *LocalCache[T]) UnLock(ctx context.Context, key string) {
	this.lock.Lock()
	defer this.lock.Unlock()

	this.Del(ctx, key)
}
