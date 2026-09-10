package util

import (
	"context"
	"strings"
	"sync"
	"time"

	"github.com/patrickmn/go-cache"
)

var cacheReqId = NewLocalCache[int]()

func TryLockReqId(ctx context.Context, reqId int64, duration time.Duration) bool {
	key := Int2Str(reqId)
	return cacheReqId.TryLock(ctx, key, duration) > 0
}

var cacheHttpBan = NewLocalCache[int]()

func GetHttpBan(ctx context.Context, address string) bool {
	address = strings.Split(address, "#")[0]
	address = strings.Split(address, "?")[0]
	key := address
	_, ok := cacheHttpBan.Get(ctx, key)
	return ok
}
func SetHttpBan(ctx context.Context, address string, duration time.Duration) {
	address = strings.Split(address, "#")[0]
	address = strings.Split(address, "?")[0]
	key := address
	cacheHttpBan.Set(ctx, key, duration, 0)
}

func NewLocalCache[T any]() LocalCache[T] {
	return LocalCache[T]{lock: &sync.RWMutex{}, cache: cache.New(time.Minute, time.Minute)}
}

type LocalCache[T any] struct {
	lock  *sync.RWMutex
	cache *cache.Cache
}

func (this *LocalCache[T]) get(ctx context.Context, key string) (T, bool) {
	var value T
	object, ok := this.cache.Get(key)
	if !ok {
		return value, false
	}
	if object == nil {
		return value, true
	}
	value, ok = object.(T)
	return value, ok
}
func (this *LocalCache[T]) Get(ctx context.Context, key string) (T, bool) {
	this.lock.RLock()
	defer this.lock.RUnlock()

	return this.get(ctx, key)
}
func (this *LocalCache[T]) set(ctx context.Context, key string, duration time.Duration, object T) {
	this.cache.Set(key, object, duration)
}
func (this *LocalCache[T]) Set(ctx context.Context, key string, duration time.Duration, object T) {
	this.lock.Lock()
	defer this.lock.Unlock()

	this.set(ctx, key, duration, object)
}
func (this *LocalCache[T]) del(ctx context.Context, key string) {
	this.cache.Delete(key)
}
func (this *LocalCache[T]) Del(ctx context.Context, key string) {
	this.lock.Lock()
	defer this.lock.Unlock()

	this.del(ctx, key)
}
func (this *LocalCache[T]) fetch(ctx context.Context, key string, duration time.Duration, get func() (T, error)) (T, error) {
	object, ok := this.get(ctx, key)
	if ok {
		return object, nil
	}
	object, err := get()
	if err != nil {
		return object, err
	}
	this.set(ctx, key, duration, object)
	return object, nil
}
func (this *LocalCache[T]) Fetch(ctx context.Context, key string, duration time.Duration, get func() (T, error)) (T, error) {
	this.lock.Lock()
	defer this.lock.Unlock()

	return this.fetch(ctx, key, duration, get)
}
func (this *LocalCache[T]) tryLock(ctx context.Context, key string, duration time.Duration) int64 {
	lockId := GenId()
	err := this.cache.Add(key, lockId, duration) //Add是"键不存在才写入"的原子操作
	if err != nil {
		return 0
	}
	return lockId
}
func (this *LocalCache[T]) TryLock(ctx context.Context, key string, duration time.Duration) int64 {
	this.lock.Lock()
	defer this.lock.Unlock()

	return this.tryLock(ctx, key, duration)
}
func (this *LocalCache[T]) unLock(ctx context.Context, key string, lockId int64) {
	object, ok := this.cache.Get(key)
	if !ok {
		return
	}
	id, ok := object.(int64)
	if !ok || id != lockId {
		return
	}
	this.cache.Delete(key)
}
func (this *LocalCache[T]) UnLock(ctx context.Context, key string, lockId int64) {
	this.lock.Lock()
	defer this.lock.Unlock()

	this.unLock(ctx, key, lockId)
}
