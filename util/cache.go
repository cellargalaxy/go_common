package util

import (
	"context"
	"fmt"
	"github.com/patrickmn/go-cache"
	"strings"
	"sync"
	"time"
)

var localCache = NewLocalCache[int]()

// existReqId 判断reqId是否已出现过，用于ValidateGin的防重放校验。
// 必须原子：此前实现是 Get 后再 Set 两步非原子操作，同一reqId被并发校验时
// 多个协程会同时读到"不存在"并一起放行（实测64协程并发下约18%的轮次有2个以上请求通过），
// 使防重放形同虚设。改用带互斥的 TryLock：
// 拿到锁 => 首次出现（返回false，放行）；拿不到锁 => 已存在（返回true，判定重放）。
func existReqId(ctx context.Context, reqId string, duration time.Duration) bool {
	key := fmt.Sprintf("reqId-%s", reqId)
	return !localCache.TryLock(ctx, key, duration)
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
	localCache.Set(ctx, key, 0, duration)
}

func NewLocalCache[T any]() LocalCache[T] {
	return LocalCache[T]{lock: &sync.Mutex{}, cache: cache.New(time.Minute, time.Minute), timeMap: make(map[string]time.Time)}
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
	//键存在但值为nil时，不能用 object.(T) 的结果当作"是否存在"：
	//对nil接口做类型断言恒为false（即便T是any），会把"存在且值为nil"误判成"不存在"。
	//该误判会让 TryLock 对 T=any 永远拿到锁（互斥失效），
	//也会让 GetWithTimeout 对nil缓存值每次都穿透重取。
	if object == nil {
		return value, true
	}
	value, ok = object.(T)
	return value, ok
}
func (this *LocalCache[T]) Set(ctx context.Context, key string, object T, duration time.Duration) {
	this.cache.Set(key, object, duration)
}
func (this *LocalCache[T]) Del(ctx context.Context, key string) {
	this.lock.Lock()
	defer this.lock.Unlock()

	this.del(ctx, key)
}

// del 不加锁，供已持锁的 Del/UnLock 复用。
// timeMap 是普通 map，必须在 this.lock 保护下访问：
// GetWithTimeout 持锁读写它，若 Del 裸写会构成数据竞争(-race 可复现)。
func (this *LocalCache[T]) del(ctx context.Context, key string) {
	this.cache.Delete(key)
	//必须同步清理timeMap：GetWithTimeout写入后由timeMap记录取值时刻，
	//只删cache会让timeMap的键永久残留，长期按动态键(如reqId、address)使用时内存持续增长
	delete(this.timeMap, key)
}
func (this *LocalCache[T]) GetWithTimeout(ctx context.Context, key string, duration time.Duration, get func() (T, error)) (T, error) {
	this.lock.Lock()
	defer this.lock.Unlock()

	object, ok := this.Get(ctx, key)
	cacheTime := this.timeMap[key]
	if ok && time.Since(cacheTime) <= duration {
		return object, nil
	}

	object, err := get()
	if err != nil {
		return object, err
	}

	this.timeMap[key] = time.Now()
	this.Set(ctx, key, object, DurationMax)

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

	//sync.Mutex不可重入，此处已持锁，只能调不加锁的del
	this.del(ctx, key)
}
