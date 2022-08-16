package util

import (
	"context"
	"fmt"
	"github.com/patrickmn/go-cache"
	"github.com/sirupsen/logrus"
	"strings"
	"sync"
	"time"
)

var localCache *LocalCache

func initCache(ctx context.Context) {
	var err error
	localCache, err = NewDefaultLocalCache(ctx)
	if err != nil {
		panic(err)
	}
}

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

func NewDefaultLocalCache(ctx context.Context) (*LocalCache, error) {
	return NewLocalCache(ctx, cache.New(time.Minute, time.Minute), &sync.Mutex{})
}
func NewLocalCache(ctx context.Context, cache *cache.Cache, lock *sync.Mutex) (*LocalCache, error) {
	if cache == nil {
		logrus.WithContext(ctx).WithFields(logrus.Fields{}).Error("创建LocalCache，cache为空")
		return nil, fmt.Errorf("创建LocalCache，cache为空")
	}
	if lock == nil {
		logrus.WithContext(ctx).WithFields(logrus.Fields{}).Error("创建LocalCache，lock为空")
		return nil, fmt.Errorf("创建LocalCache，lock为空")
	}
	return &LocalCache{cache: cache, lock: lock}, nil
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

	this.Set(ctx, key, Object{object: object, cacheTime: time.Now()}, MaxTime.Sub(time.Now()))
	return object, nil
}
