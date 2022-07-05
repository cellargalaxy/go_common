package util

import (
	"github.com/patrickmn/go-cache"
	"time"
)

var localCache *cache.Cache

func initCache() {
	localCache = cache.New(time.Minute, time.Minute)
	if localCache == nil {
		panic("创建本地缓存对象为空")
	}
}

func existRequestId(reqId string, duration time.Duration) bool {
	_, ok := localCache.Get("reqid-" + reqId)
	localCache.Set(reqId, "", duration)
	return ok
}
