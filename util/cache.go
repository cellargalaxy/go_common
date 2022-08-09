package util

import (
	"fmt"
	"github.com/patrickmn/go-cache"
	"net/url"
	"time"
)

var localCache *cache.Cache

func initCache() {
	localCache = cache.New(time.Minute, time.Minute)
	if localCache == nil {
		panic("创建本地缓存对象为空")
	}
}

func existReqId(reqId string, duration time.Duration) bool {
	key := fmt.Sprintf("reqId-%s", reqId)
	_, ok := localCache.Get(key)
	localCache.Set(key, "", duration)
	return ok
}

func getHttpBan(address string) bool {
	u, err := url.Parse(address)
	if u == nil || err != nil {
		return false
	}
	host := u.Host
	if host == "" {
		return false
	}
	key := fmt.Sprintf("httpBan-%s", host)
	_, ok := localCache.Get(key)
	return ok
}
func setHttpBan(address string, duration time.Duration) {
	u, err := url.Parse(address)
	if u == nil || err != nil {
		return
	}
	host := u.Host
	if host == "" {
		return
	}
	key := fmt.Sprintf("httpBan-%s", host)
	localCache.Set(key, "", duration)
}
