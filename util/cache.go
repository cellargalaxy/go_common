package util

import (
	"fmt"
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
	key := fmt.Sprintf("reqid-%s", reqId)
	_, ok := localCache.Get(key)
	localCache.Set(key, "", duration)
	return ok
}
