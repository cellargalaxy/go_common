package util

import (
	"sync"
	"testing"
	"time"
)

func TestLocalCacheGetSetDel(t *testing.T) {
	ctx := GenCtx()
	c := NewLocalCache[string]()

	//未设置时取不到
	if _, ok := c.Get(ctx, "nokey"); ok {
		t.Errorf("未设置的键返回了ok")
	}
	//设置后可取回
	c.Set(ctx, "k", "v", time.Hour)
	got, ok := c.Get(ctx, "k")
	if !ok || got != "v" {
		t.Errorf("Get = %q, %v", got, ok)
	}
	//覆盖
	c.Set(ctx, "k", "v2", time.Hour)
	if got, _ = c.Get(ctx, "k"); got != "v2" {
		t.Errorf("覆盖后 = %q", got)
	}
	//删除后取不到
	c.Del(ctx, "k")
	if _, ok = c.Get(ctx, "k"); ok {
		t.Errorf("删除后仍能取到")
	}
	//删除不存在的键不应panic
	c.Del(ctx, "nokey")

	//零值与空串要能被正常区分（ok标志必须可靠）
	c.Set(ctx, "empty", "", time.Hour)
	got, ok = c.Get(ctx, "empty")
	if !ok {
		t.Errorf("空串值应返回 ok=true，否则调用方无法区分'存了空串'与'没存'")
	}
	if got != "" {
		t.Errorf("空串值 = %q", got)
	}
}

// 缓存必须真的过期，否则等于永久缓存
func TestLocalCacheExpire(t *testing.T) {
	ctx := GenCtx()
	c := NewLocalCache[int]()

	c.Set(ctx, "short", 1, 50*time.Millisecond)
	if _, ok := c.Get(ctx, "short"); !ok {
		t.Fatalf("刚设置就取不到")
	}
	time.Sleep(150 * time.Millisecond)
	if _, ok := c.Get(ctx, "short"); ok {
		t.Errorf("过期后仍能取到，缓存未过期")
	}

	//长过期时间不应提前失效
	c.Set(ctx, "long", 2, time.Hour)
	time.Sleep(120 * time.Millisecond)
	if got, ok := c.Get(ctx, "long"); !ok || got != 2 {
		t.Errorf("长过期键提前失效: %d, %v", got, ok)
	}
}

// 泛型缓存：类型不匹配时不能panic，只能返回false
func TestLocalCacheTypeMismatch(t *testing.T) {
	ctx := GenCtx()
	c := NewLocalCache[string]()
	//借用底层cache塞入一个int，模拟类型不一致
	c.cache.Set("bad", 123, time.Hour)
	got, ok := c.Get(ctx, "bad")
	if ok {
		t.Errorf("类型不匹配应返回 false, got %q", got)
	}
	if got != "" {
		t.Errorf("类型不匹配应返回零值, got %q", got)
	}
}

// 结构体值拷贝共享同一份底层缓存（原用例依赖此行为，需保持）
func TestLocalCacheShareByCopy(t *testing.T) {
	ctx := GenCtx()
	c1 := NewLocalCache[string]()
	c2 := c1

	c1.Set(ctx, "k", "v", time.Hour)
	if got, ok := c2.Get(ctx, "k"); !ok || got != "v" {
		t.Errorf("拷贝后未共享底层缓存: %q, %v", got, ok)
	}
	//锁也必须共享（lock 为指针）
	if c1.lock != c2.lock {
		t.Errorf("拷贝后锁未共享，并发保护会失效")
	}
	c2.Del(ctx, "k")
	if _, ok := c1.Get(ctx, "k"); ok {
		t.Errorf("通过副本删除后原对象仍能取到")
	}
}

func TestLocalCacheGetWithTimeout(t *testing.T) {
	ctx := GenCtx()
	c := NewLocalCache[string]()

	//首次调用须执行get并缓存
	calls := 0
	got, err := c.GetWithTimeout(ctx, "k", time.Hour, func() (string, error) {
		calls++
		return "first", nil
	})
	if err != nil || got != "first" {
		t.Fatalf("首次 = %q, %v", got, err)
	}
	if calls != 1 {
		t.Errorf("get 调用次数 = %d, 期望 1", calls)
	}
	//缓存期内不得再次执行get
	got, err = c.GetWithTimeout(ctx, "k", time.Hour, func() (string, error) {
		calls++
		return "second", nil
	})
	if err != nil {
		t.Fatalf("%+v", err)
	}
	if got != "first" {
		t.Errorf("缓存期内 = %q, 期望仍为 first", got)
	}
	if calls != 1 {
		t.Errorf("缓存期内 get 被重复调用, 次数 = %d", calls)
	}

	//超过duration后须重新取值
	got, err = c.GetWithTimeout(ctx, "k", time.Nanosecond, func() (string, error) {
		calls++
		return "refreshed", nil
	})
	if err != nil {
		t.Fatalf("%+v", err)
	}
	if got != "refreshed" {
		t.Errorf("超时后 = %q, 期望 refreshed", got)
	}
	if calls != 2 {
		t.Errorf("超时后 get 调用次数 = %d, 期望 2", calls)
	}
}

// get 返回错误时不得写入缓存，也不得吞掉错误
func TestLocalCacheGetWithTimeoutError(t *testing.T) {
	ctx := GenCtx()
	c := NewLocalCache[string]()

	wantErr := ErrTestSentinel
	got, err := c.GetWithTimeout(ctx, "k", time.Hour, func() (string, error) {
		return "ignored", wantErr
	})
	if err == nil {
		t.Fatalf("get 报错时应返回error")
	}
	if got != "ignored" {
		t.Logf("失败时返回值 = %q（当前实现原样透传get的返回值）", got)
	}
	//失败不得写缓存，否则错误会被固化
	if _, ok := c.Get(ctx, "k"); ok {
		t.Errorf("get 失败后仍写入了缓存")
	}
	//下一次调用必须重新尝试
	calls := 0
	got, err = c.GetWithTimeout(ctx, "k", time.Hour, func() (string, error) {
		calls++
		return "ok", nil
	})
	if err != nil || got != "ok" || calls != 1 {
		t.Errorf("失败后重试异常: %q, %v, calls=%d", got, err, calls)
	}
}

func TestLocalCacheTryLock(t *testing.T) {
	ctx := GenCtx()
	c := NewLocalCache[int]()

	//首次能拿到
	if !c.TryLock(ctx, "lk", time.Hour) {
		t.Fatalf("首次 TryLock 失败")
	}
	//重复获取必须失败（互斥）
	if c.TryLock(ctx, "lk", time.Hour) {
		t.Errorf("同一key重复 TryLock 应失败")
	}
	//不同key互不影响
	if !c.TryLock(ctx, "other", time.Hour) {
		t.Errorf("不同key的 TryLock 应成功")
	}
	//释放后可再次获取
	c.UnLock(ctx, "lk")
	if !c.TryLock(ctx, "lk", time.Hour) {
		t.Errorf("UnLock 后应能重新获取")
	}
	//释放未持有的锁不应panic
	c.UnLock(ctx, "never-locked")
}

// 锁必须随过期自动释放，避免死锁
func TestLocalCacheTryLockExpire(t *testing.T) {
	ctx := GenCtx()
	c := NewLocalCache[int]()

	if !c.TryLock(ctx, "k", 50*time.Millisecond) {
		t.Fatalf("首次 TryLock 失败")
	}
	if c.TryLock(ctx, "k", time.Hour) {
		t.Errorf("未过期时应拿不到锁")
	}
	time.Sleep(150 * time.Millisecond)
	if !c.TryLock(ctx, "k", time.Hour) {
		t.Errorf("锁过期后应能重新获取，否则会死锁")
	}
}

// Get 必须区分"键不存在"与"键存在但值为nil"。
// 对nil接口做类型断言恒为false（即便目标类型是any），
// 若用断言结果当作存在性判断，会把已缓存的nil误报成未命中。
func TestLocalCacheGetNilValue(t *testing.T) {
	ctx := GenCtx()

	//T=any 且值为nil：必须报告命中
	ic := NewLocalCache[any]()
	ic.Set(ctx, "k", nil, time.Hour)
	got, ok := ic.Get(ctx, "k")
	if !ok {
		t.Errorf("LocalCache[any] 缓存nil后 Get 应命中，实际未命中")
	}
	if got != nil {
		t.Errorf("LocalCache[any] 缓存nil后取回 = %v, 期望 nil", got)
	}
	//未设置的键仍须报告未命中，不能因为上面的修改而恒为true
	if _, ok = ic.Get(ctx, "absent"); ok {
		t.Errorf("未设置的键不应命中")
	}
	//删除后必须回到未命中
	ic.Del(ctx, "k")
	if _, ok = ic.Get(ctx, "k"); ok {
		t.Errorf("删除后仍命中")
	}

	//指针类型缓存nil同样须命中
	pc := NewLocalCache[*int]()
	pc.Set(ctx, "p", nil, time.Hour)
	pv, pok := pc.Get(ctx, "p")
	if !pok {
		t.Errorf("LocalCache[*int] 缓存nil后 Get 应命中")
	}
	if pv != nil {
		t.Errorf("LocalCache[*int] 缓存nil后取回 = %v, 期望 nil", pv)
	}

	//map/slice 等可为nil的类型
	mc := NewLocalCache[map[string]int]()
	mc.Set(ctx, "m", nil, time.Hour)
	if _, ok = mc.Get(ctx, "m"); !ok {
		t.Errorf("LocalCache[map] 缓存nil后 Get 应命中")
	}
}

// TryLock 对 T=any 必须真正互斥。
// 其零值是nil接口，若 Get 用类型断言判断存在性，
// 存进去的nil永远读不出来，导致每次都能拿到锁、互斥彻底失效。
func TestLocalCacheTryLockAnyType(t *testing.T) {
	ctx := GenCtx()
	ic := NewLocalCache[any]()

	if !ic.TryLock(ctx, "lk", time.Hour) {
		t.Fatalf("LocalCache[any] 首次 TryLock 失败")
	}
	if ic.TryLock(ctx, "lk", time.Hour) {
		t.Errorf("LocalCache[any] 重复 TryLock 应失败，互斥失效")
	}
	//连续多次都不能再拿到
	for i := 0; i < 3; i++ {
		if ic.TryLock(ctx, "lk", time.Hour) {
			t.Fatalf("LocalCache[any] 第%d次重复 TryLock 仍成功", i+2)
		}
	}
	//释放后可重新获取
	ic.UnLock(ctx, "lk")
	if !ic.TryLock(ctx, "lk", time.Hour) {
		t.Errorf("LocalCache[any] UnLock 后应能重新获取")
	}

	//指针类型同理
	pc := NewLocalCache[*int]()
	if !pc.TryLock(ctx, "lk", time.Hour) {
		t.Fatalf("LocalCache[*int] 首次 TryLock 失败")
	}
	if pc.TryLock(ctx, "lk", time.Hour) {
		t.Errorf("LocalCache[*int] 重复 TryLock 应失败")
	}

	//并发场景下 T=any 也只能有一个成功
	ac := NewLocalCache[any]()
	const n = 50
	var wg sync.WaitGroup
	var mu sync.Mutex
	success := 0
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if ac.TryLock(ctx, "race", time.Hour) {
				mu.Lock()
				success++
				mu.Unlock()
			}
		}()
	}
	wg.Wait()
	if success != 1 {
		t.Errorf("LocalCache[any] 并发 TryLock 成功 %d 次, 期望 1 次", success)
	}
}

// GetWithTimeout 缓存到nil值时不能每次都穿透重取
func TestLocalCacheGetWithTimeoutNilValue(t *testing.T) {
	ctx := GenCtx()
	ic := NewLocalCache[any]()

	calls := 0
	getter := func() (any, error) {
		calls++
		return nil, nil
	}
	if _, err := ic.GetWithTimeout(ctx, "k", time.Hour, getter); err != nil {
		t.Fatalf("GetWithTimeout 异常: %+v", err)
	}
	if _, err := ic.GetWithTimeout(ctx, "k", time.Hour, getter); err != nil {
		t.Fatalf("GetWithTimeout 异常: %+v", err)
	}
	if calls != 1 {
		t.Errorf("GetWithTimeout 对nil缓存值调用了 %d 次get, 期望 1 次（nil也应命中缓存）", calls)
	}
}

// 并发下 TryLock 必须只有一个成功，且不得出现数据竞争
func TestLocalCacheTryLockConcurrent(t *testing.T) {
	ctx := GenCtx()
	c := NewLocalCache[int]()

	const n = 50
	var wg sync.WaitGroup
	var mu sync.Mutex
	success := 0
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if c.TryLock(ctx, "race-key", time.Hour) {
				mu.Lock()
				success++
				mu.Unlock()
			}
		}()
	}
	wg.Wait()
	if success != 1 {
		t.Errorf("并发 TryLock 成功次数 = %d, 期望恰好 1", success)
	}
}

// 并发 GetWithTimeout 时 get 应只被执行一次（有锁保护）
func TestLocalCacheGetWithTimeoutConcurrent(t *testing.T) {
	ctx := GenCtx()
	c := NewLocalCache[int]()

	const n = 30
	var wg sync.WaitGroup
	var mu sync.Mutex
	calls := 0
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			c.GetWithTimeout(ctx, "k", time.Hour, func() (int, error) {
				mu.Lock()
				calls++
				mu.Unlock()
				time.Sleep(time.Millisecond)
				return 1, nil
			})
		}()
	}
	wg.Wait()
	if calls != 1 {
		t.Errorf("并发下 get 执行次数 = %d, 期望 1（缓存击穿保护失效）", calls)
	}
}

// 并发 Get/Set/Del 不得触发 race 或 panic
func TestLocalCacheConcurrentAccess(t *testing.T) {
	ctx := GenCtx()
	c := NewLocalCache[int]()

	//并发过程中若发生panic，须显式失败而不是被goroutine吞掉
	panicCh := make(chan interface{}, 64)
	safe := func(fn func()) {
		defer func() {
			if r := recover(); r != nil {
				panicCh <- r
			}
		}()
		fn()
	}

	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(3)
		go func(n int) { defer wg.Done(); safe(func() { c.Set(ctx, "k", n, time.Minute) }) }(i)
		go func() { defer wg.Done(); safe(func() { c.Get(ctx, "k") }) }()
		go func() { defer wg.Done(); safe(func() { c.Del(ctx, "k") }) }()
	}
	wg.Wait()
	close(panicCh)
	for r := range panicCh {
		t.Errorf("并发访问触发panic: %v", r)
	}

	//并发结束后缓存必须仍可正常读写，证明内部状态没被并发破坏
	c.Set(ctx, "after", 42, time.Minute)
	if got, exist := c.Get(ctx, "after"); !exist || got != 42 {
		t.Errorf("并发后缓存不可用: got=%v exist=%v", got, exist)
	}
	c.Del(ctx, "after")
	if _, exist := c.Get(ctx, "after"); exist {
		t.Errorf("并发后 Del 失效")
	}
}

// ==== 包内私有缓存函数 ====

func TestExistReqId(t *testing.T) {
	ctx := GenCtx()
	reqId := GenStringId()

	//首次未出现过
	if existReqId(ctx, reqId, time.Hour) {
		t.Errorf("首个reqId应返回 false")
	}
	//第二次必须判定为已存在（幂等去重的核心）
	if !existReqId(ctx, reqId, time.Hour) {
		t.Errorf("重复reqId应返回 true")
	}
	//不同reqId互不影响
	if existReqId(ctx, GenStringId(), time.Hour) {
		t.Errorf("不同reqId应返回 false")
	}
	//过期后视为未出现
	shortId := GenStringId()
	existReqId(ctx, shortId, 50*time.Millisecond)
	time.Sleep(150 * time.Millisecond)
	if existReqId(ctx, shortId, time.Hour) {
		t.Errorf("过期后应重新视为未出现")
	}
}

func TestHttpBan(t *testing.T) {
	ctx := GenCtx()
	address := "http://" + GenStringId() + ".com/path"

	//默认未封禁
	if getHttpBan(ctx, address) {
		t.Errorf("默认应未封禁")
	}
	//设置后封禁
	setHttpBan(ctx, address, time.Hour)
	if !getHttpBan(ctx, address) {
		t.Errorf("设置后应为封禁")
	}
	//query 与 fragment 须被忽略：同一地址的不同参数应共享封禁状态
	if !getHttpBan(ctx, address+"?a=1") {
		t.Errorf("带query的同地址应同样被封禁")
	}
	if !getHttpBan(ctx, address+"#frag") {
		t.Errorf("带fragment的同地址应同样被封禁")
	}
	//不同地址不受影响
	if getHttpBan(ctx, "http://"+GenStringId()+".com/other") {
		t.Errorf("其他地址不应被封禁")
	}
	//过期后解封
	shortAddr := "http://" + GenStringId() + ".com"
	setHttpBan(ctx, shortAddr, 50*time.Millisecond)
	time.Sleep(150 * time.Millisecond)
	if getHttpBan(ctx, shortAddr) {
		t.Errorf("过期后应自动解封")
	}
}
