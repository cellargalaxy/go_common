package util

import (
	"context"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestNewSingleGoPool(t *testing.T) {
	ctx := GenCtx()
	pool, err := NewSingleGoPool(ctx, "mypool")
	if err != nil {
		t.Fatalf("NewSingleGoPool 异常: %+v", err)
	}
	if pool == nil {
		t.Fatalf("返回 nil 池")
	}
	defer ClosePool(ctx, pool)

	if got := pool.GetPoolName(); got != "mypool" {
		t.Errorf("GetPoolName = %q, 期望 mypool", got)
	}
	//新建池未跑任务，任务名为空、非Doing
	if got := pool.GetTaskName(); got != "" {
		t.Errorf("新建池 GetTaskName = %q, 期望空", got)
	}
	if pool.Doing(ctx) {
		t.Errorf("新建池不应处于 Doing")
	}
	if pool.IsClose(ctx) {
		t.Errorf("新建池不应已关闭")
	}
}

// GetName 组合规则：poolName_taskName，任一为空时退化
func TestSingleGoPoolGetName(t *testing.T) {
	ctx := GenCtx()

	//两者都有
	pool, err := NewSingleGoPool(ctx, "P")
	if err != nil {
		t.Fatalf("%+v", err)
	}
	defer ClosePool(ctx, pool)
	pool.setTaskName("T")
	if got := pool.GetName(); got != "P_T" {
		t.Errorf("GetName = %q, 期望 P_T", got)
	}
	//只有池名
	pool.setTaskName("")
	if got := pool.GetName(); got != "P" {
		t.Errorf("GetName = %q, 期望 P", got)
	}

	//只有任务名
	anon, err := NewSingleGoPool(ctx, "")
	if err != nil {
		t.Fatalf("%+v", err)
	}
	defer ClosePool(ctx, anon)
	anon.setTaskName("T")
	if got := anon.GetName(); got != "T" {
		t.Errorf("GetName = %q, 期望 T", got)
	}
	//都为空时用兜底名
	anon.setTaskName("")
	if got := anon.GetName(); got != "SingleGoPool" {
		t.Errorf("GetName = %q, 期望 SingleGoPool", got)
	}
}

// NewOnceSingleGoPool 传空poolName是有意设计，避免出现 MyTask_MyTask
func TestNewOnceSingleGoPoolName(t *testing.T) {
	ctx := GenCtx()
	var started atomic.Bool
	var nameInTask atomic.Value
	nameInTask.Store("")

	pool, err := NewOnceSingleGoPool(ctx, "MyTask", func(cancelCtx context.Context, pool *SingleGoPool) {
		nameInTask.Store(pool.GetName())
		started.Store(true)
	})
	if err != nil {
		t.Fatalf("%+v", err)
	}
	defer ClosePool(ctx, pool)

	waitFor(t, 2000, func() bool { return started.Load() })
	//池名为空、任务名为MyTask，故GetName应恰为MyTask而非MyTask_MyTask
	if got, _ := nameInTask.Load().(string); got != "MyTask" {
		t.Errorf("任务内 GetName = %q, 期望 MyTask（不应重复成 MyTask_MyTask）", got)
	}
	if got := pool.GetPoolName(); got != "" {
		t.Errorf("OnceSingleGoPool 的 poolName = %q, 期望空", got)
	}
}

// 空任务名须自动生成，而不是让任务名为空导致后续无法按名字识别任务
func TestSingleGoPoolEmptyTaskName(t *testing.T) {
	ctx := GenCtx()
	pool, err := NewSingleGoPool(ctx, "P")
	if err != nil {
		t.Fatalf("%+v", err)
	}
	defer ClosePool(ctx, pool)

	if err = pool.AddOnceTask(ctx, "", func(context.Context, *SingleGoPool) {}); err != nil {
		t.Errorf("空任务名的 AddOnceTask 异常: %+v", err)
	}
	if got := pool.GetTaskName(); got == "" {
		t.Errorf("空任务名未被自动生成")
	}

	pool.Cancel(ctx)
	if err = pool.AddDaemonTask(ctx, "", time.Millisecond, func(context.Context, *SingleGoPool) {}); err != nil {
		t.Errorf("空任务名的 AddDaemonTask 异常: %+v", err)
	}
	if got := pool.GetTaskName(); got == "" {
		t.Errorf("空任务名未被自动生成")
	}
}

// 单次任务：新任务替换旧任务，且旧任务的cancelCtx被取消
func TestOnceSingleGoPool(t *testing.T) {
	ctx := GenCtx()

	//taskChange由任务协程写、测试协程读，须原子访问，否则-race下必然报竞争
	var taskChange atomic.Value
	taskChange.Store("")
	getTaskChange := func() string {
		value, _ := taskChange.Load().(string)
		return value
	}
	pool, err := NewOnceSingleGoPool(ctx, "test-1", func(cancelCtx context.Context, pool *SingleGoPool) {
		for {
			taskChange.Store("test-1")
			time.Sleep(time.Millisecond * 50)
			if CtxDone(cancelCtx) {
				return
			}
		}
	})
	if err != nil {
		t.Fatalf("%+v", err)
	}
	defer ClosePool(ctx, pool)

	//第一个任务须真正跑起来
	waitFor(t, 3000, func() bool { return getTaskChange() == "test-1" })
	if !pool.Doing(ctx) {
		t.Errorf("任务运行中 Doing 应为 true")
	}
	if got := pool.GetTaskName(); got != "test-1" {
		t.Errorf("GetTaskName = %q, 期望 test-1", got)
	}

	//加入第二个任务：应取消第一个并接管
	if err = pool.AddOnceTask(ctx, "test-2", func(cancelCtx context.Context, pool *SingleGoPool) {
		for {
			taskChange.Store("test-2")
			time.Sleep(time.Millisecond * 50)
			if CtxDone(cancelCtx) {
				taskChange.Store("test-done")
				return
			}
		}
	}); err != nil {
		t.Fatalf("AddOnceTask 异常: %+v", err)
	}
	waitFor(t, 3000, func() bool { return getTaskChange() == "test-2" })

	//关闭池：任务须感知cancelCtx并优雅退出
	ClosePool(ctx, pool)
	waitFor(t, 3000, func() bool { return getTaskChange() == "test-done" })
	if !pool.IsClose(ctx) {
		t.Errorf("关闭后 IsClose 应为 true")
	}
	if pool.Doing(ctx) {
		t.Errorf("关闭后 Doing 应为 false")
	}
}

// 重复添加同名任务应被忽略，不能重复启动
func TestOnceSingleGoPoolSameName(t *testing.T) {
	ctx := GenCtx()
	var runs atomic.Int64
	pool, err := NewOnceSingleGoPool(ctx, "same", func(cancelCtx context.Context, pool *SingleGoPool) {
		runs.Add(1)
		for !CtxDone(cancelCtx) {
			time.Sleep(time.Millisecond * 20)
		}
	})
	if err != nil {
		t.Fatalf("%+v", err)
	}
	defer ClosePool(ctx, pool)

	waitFor(t, 2000, func() bool { return runs.Load() == 1 })
	//同名任务重复添加：应直接返回nil且不新增执行
	for i := 0; i < 3; i++ {
		if err = pool.AddOnceTask(ctx, "same", func(context.Context, *SingleGoPool) { runs.Add(1) }); err != nil {
			t.Errorf("同名任务应被忽略而非报错: %+v", err)
		}
	}
	time.Sleep(200 * time.Millisecond)
	if got := runs.Load(); got != 1 {
		t.Errorf("同名任务执行次数 = %d, 期望 1", got)
	}
}

// 守护任务：panic后须自动重启（本用例用除零panic验证）
func TestDaemonSingleGoPool(t *testing.T) {
	ctx := GenCtx()

	//i由任务协程写、测试协程读写，须原子访问；注意 100/i 的除零panic是本用例故意用来验证守护重启的
	var i atomic.Int64
	i.Store(3)
	var panics atomic.Int64
	pool, err := NewDaemonSingleGoPool(ctx, "test", time.Millisecond*50, func(cancelCtx context.Context, pool *SingleGoPool) {
		for {
			current := i.Add(-1)
			if current == 0 {
				panics.Add(1)
			}
			//current为0时触发除零panic，守护逻辑应捕获并重启任务
			_ = 100 / current
			time.Sleep(time.Millisecond * 50)
			if CtxDone(cancelCtx) {
				return
			}
		}
	})
	if err != nil {
		t.Fatalf("%+v", err)
	}

	//i 必须被减到负数，说明panic后任务确实被重启并继续执行
	waitFor(t, 5000, func() bool { return i.Load() < 0 })
	if panics.Load() < 1 {
		t.Errorf("未触发预期的panic，用例未验证到守护重启")
	}

	//关闭后不得再重启：i 不应再被改动
	ClosePool(ctx, pool)
	time.Sleep(300 * time.Millisecond)
	i.Store(10000)
	time.Sleep(500 * time.Millisecond)
	if got := i.Load(); got != 10000 {
		t.Errorf("关闭后任务仍在运行, i = %d, 期望保持 10000", got)
	}
	if !pool.IsClose(ctx) {
		t.Errorf("关闭后 IsClose 应为 true")
	}
}

// Cancel 只取消当前任务，池仍可继续接受新任务；Close 则彻底关闭
func TestSingleGoPoolCancelVsClose(t *testing.T) {
	ctx := GenCtx()
	pool, err := NewSingleGoPool(ctx, "P")
	if err != nil {
		t.Fatalf("%+v", err)
	}
	defer ClosePool(ctx, pool)

	var stopped atomic.Bool
	if err = pool.AddOnceTask(ctx, "t1", func(cancelCtx context.Context, pool *SingleGoPool) {
		for !CtxDone(cancelCtx) {
			time.Sleep(time.Millisecond * 20)
		}
		stopped.Store(true)
	}); err != nil {
		t.Fatalf("%+v", err)
	}
	waitFor(t, 2000, func() bool { return pool.Doing(ctx) })

	//Cancel：任务应退出，但池未关闭
	pool.Cancel(ctx)
	waitFor(t, 3000, func() bool { return stopped.Load() })
	if pool.IsClose(ctx) {
		t.Errorf("Cancel 不应关闭池")
	}
	if pool.Doing(ctx) {
		t.Errorf("Cancel 后 Doing 应为 false")
	}
	//取消后仍可添加新任务
	var second atomic.Bool
	if err = pool.AddOnceTask(ctx, "t2", func(cancelCtx context.Context, pool *SingleGoPool) {
		second.Store(true)
	}); err != nil {
		t.Errorf("Cancel 后应仍能添加任务: %+v", err)
	}
	waitFor(t, 3000, func() bool { return second.Load() })

	//Close 后池关闭
	pool.Close(ctx)
	if !pool.IsClose(ctx) {
		t.Errorf("Close 后 IsClose 应为 true")
	}
}

// CancelPool / ClosePool 批量操作，须跳过nil且可重复调用
func TestCancelAndClosePool(t *testing.T) {
	ctx := GenCtx()
	//nil 与空入参不能panic
	CancelPool(ctx, nil)
	CancelPool(ctx)
	ClosePool(ctx, nil)
	ClosePool(ctx)

	p1, err := NewSingleGoPool(ctx, "p1")
	if err != nil {
		t.Fatalf("%+v", err)
	}
	p2, err := NewSingleGoPool(ctx, "p2")
	if err != nil {
		t.Fatalf("%+v", err)
	}
	//批量取消，含nil
	CancelPool(ctx, p1, nil, p2)
	if p1.IsClose(ctx) || p2.IsClose(ctx) {
		t.Errorf("CancelPool 不应关闭池")
	}
	//批量关闭，含nil
	ClosePool(ctx, p1, nil, p2)
	if !p1.IsClose(ctx) || !p2.IsClose(ctx) {
		t.Errorf("ClosePool 后两个池都应关闭")
	}
	//重复关闭不能panic
	ClosePool(ctx, p1, p2)
}

// 已关闭的池添加任务不得panic，任务也不应执行
func TestSingleGoPoolAddAfterClose(t *testing.T) {
	ctx := GenCtx()
	pool, err := NewSingleGoPool(ctx, "P")
	if err != nil {
		t.Fatalf("%+v", err)
	}
	pool.Close(ctx)

	var ran atomic.Bool
	//约定：已关闭时直接返回nil而非报错，但任务不得执行
	if err = pool.AddOnceTask(ctx, "t", func(context.Context, *SingleGoPool) { ran.Store(true) }); err != nil {
		t.Logf("关闭后添加任务返回: %+v", err)
	}
	time.Sleep(200 * time.Millisecond)
	if ran.Load() {
		t.Errorf("已关闭的池仍执行了任务")
	}
}

// ctx 取消后添加任务不应真正执行
func TestSingleGoPoolCtxDone(t *testing.T) {
	ctx := GenCtx()
	pool, err := NewSingleGoPool(ctx, "P")
	if err != nil {
		t.Fatalf("%+v", err)
	}
	defer ClosePool(ctx, pool)

	cancelledCtx, cancel := context.WithCancel(ctx)
	cancel()
	var ran atomic.Bool
	if err = pool.AddOnceTask(cancelledCtx, "t", func(context.Context, *SingleGoPool) { ran.Store(true) }); err != nil {
		t.Logf("已取消ctx添加任务返回: %+v", err)
	}
	time.Sleep(200 * time.Millisecond)
	if ran.Load() {
		t.Errorf("已取消的ctx仍执行了任务")
	}
}

// 任务内部可通过 pool 参数访问自身状态，且日志取名不得死锁
func TestSingleGoPoolTaskSelfAccess(t *testing.T) {
	ctx := GenCtx()
	var doing atomic.Bool
	var name atomic.Value
	name.Store("")
	var done atomic.Bool

	pool, err := NewOnceSingleGoPool(ctx, "selftask", func(cancelCtx context.Context, p *SingleGoPool) {
		//运行中在任务内读取状态：若取名实现改回加锁会在此死锁
		doing.Store(p.Doing(cancelCtx))
		name.Store(p.GetTaskName())
		p.IsClose(cancelCtx)
		p.GetName()
		done.Store(true)
	})
	if err != nil {
		t.Fatalf("%+v", err)
	}
	defer ClosePool(ctx, pool)

	//若发生死锁，此处会超时失败（历史上曾挂死91秒）
	waitFor(t, 5000, func() bool { return done.Load() })
	if !doing.Load() {
		t.Errorf("任务内 Doing 应为 true")
	}
	if got, _ := name.Load().(string); got != "selftask" {
		t.Errorf("任务内 GetTaskName = %q, 期望 selftask", got)
	}
}

// 并发调用只读方法不得触发 race
func TestSingleGoPoolConcurrentRead(t *testing.T) {
	ctx := GenCtx()
	pool, err := NewSingleGoPool(ctx, "P")
	if err != nil {
		t.Fatalf("%+v", err)
	}
	defer ClosePool(ctx, pool)

	if err = pool.AddOnceTask(ctx, "t", func(cancelCtx context.Context, pool *SingleGoPool) {
		for !CtxDone(cancelCtx) {
			time.Sleep(time.Millisecond * 10)
		}
	}); err != nil {
		t.Fatalf("%+v", err)
	}

	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func() {
			for j := 0; j < 50; j++ {
				pool.Doing(ctx)
				pool.GetTaskName()
				pool.GetName()
				pool.GetPoolName()
				pool.IsClose(ctx)
			}
			done <- true
		}()
	}
	for i := 0; i < 10; i++ {
		select {
		case <-done:
		case <-timeAfterMs(10000):
			t.Fatalf("并发读取超时，疑似死锁")
		}
	}
	pool.Cancel(ctx)
}

// waitFor 轮询等待条件成立，超时则失败；避免用固定sleep造成偶发失败
func waitFor(t *testing.T, timeoutMs int, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(time.Duration(timeoutMs) * time.Millisecond)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("等待条件超时(%dms)，调用位置见堆栈", timeoutMs)
}

// 校验守护任务名与池名的拼接（守护池会带上池名）
func TestNewDaemonSingleGoPoolName(t *testing.T) {
	ctx := GenCtx()
	var name atomic.Value
	name.Store("")
	var done atomic.Bool

	pool, err := NewDaemonSingleGoPool(ctx, "daemon", time.Hour, func(cancelCtx context.Context, p *SingleGoPool) {
		name.Store(p.GetName())
		done.Store(true)
		for !CtxDone(cancelCtx) {
			time.Sleep(time.Millisecond * 20)
		}
	})
	if err != nil {
		t.Fatalf("%+v", err)
	}
	defer ClosePool(ctx, pool)

	waitFor(t, 3000, func() bool { return done.Load() })
	got, _ := name.Load().(string)
	//守护池的poolName为空，故名称应为任务名
	if !strings.Contains(got, "daemon") {
		t.Errorf("任务内 GetName = %q, 应包含 daemon", got)
	}
}

// 守护任务的同名去重守卫：AddDaemonTask 对已在跑的同名任务须直接返回nil且不重复提交。
// 此前仅 AddOnceTask 有同名用例，守护侧的 if this.loadTaskName() == name 分支无人覆盖。
func TestDaemonSingleGoPoolSameName(t *testing.T) {
	ctx := GenCtx()
	var starts atomic.Int64
	//sleep给足够长，确保重启逻辑不会干扰计数；任务常驻直到ctx取消
	pool, err := NewDaemonSingleGoPool(ctx, "same-daemon", time.Hour, func(cancelCtx context.Context, p *SingleGoPool) {
		starts.Add(1)
		for !CtxDone(cancelCtx) {
			time.Sleep(time.Millisecond * 20)
		}
	})
	if err != nil {
		t.Fatalf("%+v", err)
	}
	defer ClosePool(ctx, pool)

	waitFor(t, 3000, func() bool { return starts.Load() == 1 })

	//关键：同名守护任务重复添加，必须被去重守卫拦下——既不报错，也不重新提交导致任务重启
	for i := 0; i < 3; i++ {
		if err = pool.AddDaemonTask(ctx, "same-daemon", time.Hour, func(context.Context, *SingleGoPool) {
			starts.Add(1)
		}); err != nil {
			t.Errorf("同名守护任务应被忽略而非报错: %+v", err)
		}
	}
	time.Sleep(200 * time.Millisecond)
	if got := starts.Load(); got != 1 {
		t.Errorf("同名守护任务启动次数 = %d, 期望 1（去重守卫失效会重启任务）", got)
	}
	//任务名不应被后续同名添加改写
	if got := pool.GetTaskName(); got != "same-daemon" {
		t.Errorf("GetTaskName = %q, 期望 same-daemon", got)
	}
	if !pool.Doing(ctx) {
		t.Errorf("守护任务仍在跑，Doing 应为 true")
	}
}
