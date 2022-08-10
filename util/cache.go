package util

import (
	"fmt"
	"github.com/patrickmn/go-cache"
	"strings"
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
	address = strings.Split(address, "#")[0]
	address = strings.Split(address, "?")[0]
	key := fmt.Sprintf("httpBan-%s", address)
	_, ok := localCache.Get(key)
	return ok
}
func setHttpBan(address string, duration time.Duration) {
	address = strings.Split(address, "#")[0]
	address = strings.Split(address, "?")[0]
	key := fmt.Sprintf("httpBan-%s", address)
	localCache.Set(key, "", duration)
}
