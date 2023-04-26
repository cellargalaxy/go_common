package main

import (
	"fmt"
	"github.com/patrickmn/go-cache"
	"sync"
	"time"
)

func init() {
	//util.Init("go_common")
}

func main() {
	//ctx := util.GenCtx()
	//fmt.Println(util.GetLogIdString(ctx))
	//logrus.WithContext(ctx).WithFields(logrus.Fields{}).Info("打印日志")

	type LocalCache struct {
		cache *cache.Cache
		lock  *sync.Mutex
	}

	o1 := LocalCache{}
	o1.cache = cache.New(time.Minute, time.Minute)
	o1.lock = &sync.Mutex{}
	o2 := o1

	fmt.Println("111")
	fmt.Println(o1.lock.TryLock())
	o1.cache.Set("aaa", "aaa", time.Hour)
	fmt.Println("222")
	fmt.Println(o2.lock.TryLock())
	fmt.Println(o2.cache.Get("aaa"))
	fmt.Println("333")
}
