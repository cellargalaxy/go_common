package util

import (
	"context"
	"fmt"
	"github.com/patrickmn/go-cache"
	"strings"
	"sync"
	"time"
)

var localCache = NewLocalCache()

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

func NewLocalCache() *LocalCache {
	return &LocalCache{cache: cache.New(time.Minute, time.Minute), lock: &sync.Mutex{}}
}

type LocalCache struct {
	cache *cache.Cache
	lock  *sync.Mutex
}

func (this *LocalCache) Get(ctx context.Context, key string) (interface{}, bool) {
	return this.cache.Get(key)
}
func (this *LocalCache) Set(ctx context.Context, key string, object interface{}, duration time.Duration) {
	this.cache.Set(key, object, duration)
}
func (this *LocalCache) Del(ctx context.Context, key string) {
	this.cache.Delete(key)
}
func (this *LocalCache) GetWithTimeout(ctx context.Context, key string, duration time.Duration, get func() (interface{}, error)) (interface{}, error) {
	this.lock.Lock()
	defer this.lock.Unlock()

	type Object struct {
		object    interface{}
		cacheTime time.Time
	}

	var obj Object
	object, ok := this.Get(ctx, key)
	if object != nil && ok {
		obj, ok = object.(Object)
		if ok && time.Now().Sub(obj.cacheTime) <= duration {
			return obj.object, nil
		}
	}

	object, err := get()
	if err != nil {
		return obj.object, err
	}
	if object == nil {
		return obj.object, nil
	}

	this.Set(ctx, key, Object{object: object, cacheTime: time.Now()}, DurationMax)
	return object, nil
}

// true:拿到锁；false:拿不到锁
func (this *LocalCache) TryLock(ctx context.Context, key string, duration time.Duration) bool {
	this.lock.Lock()
	defer this.lock.Unlock()

	_, ok := this.Get(ctx, key)
	if ok {
		return false
	}

	this.Set(ctx, key, struct{}{}, duration)
	return true
}
func (this *LocalCache) UnLock(ctx context.Context, key string) {
	this.lock.Lock()
	defer this.lock.Unlock()

	this.Del(ctx, key)
}
