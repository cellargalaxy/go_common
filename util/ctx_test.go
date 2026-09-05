package util

import (
	"context"
	"testing"
)

func TestGetSetCtxValue(t *testing.T) {
	ctx := context.Background()
	//未设置时返回类型零值，不能panic
	if got := GetCtxValue[string](ctx, "nokey"); got != "" {
		t.Errorf("未设置的string键 = %q, 期望空串", got)
	}
	if got := GetCtxValue[int](ctx, "nokey"); got != 0 {
		t.Errorf("未设置的int键 = %d", got)
	}
	//设置后可取回
	ctx = SetCtxValue(ctx, "k", "v")
	if got := GetCtxValue[string](ctx, "k"); got != "v" {
		t.Errorf("GetCtxValue = %q", got)
	}
	//类型不匹配时返回零值而非panic
	if got := GetCtxValue[int](ctx, "k"); got != 0 {
		t.Errorf("类型不匹配应返回零值, got %d", got)
	}
	//结构体值
	type payload struct{ N int }
	ctx = SetCtxValue(ctx, "p", payload{N: 7})
	if got := GetCtxValue[payload](ctx, "p"); got.N != 7 {
		t.Errorf("结构体取值 = %v", got)
	}
	//覆盖同名键，取到最新值
	ctx = SetCtxValue(ctx, "k", "v2")
	if got := GetCtxValue[string](ctx, "k"); got != "v2" {
		t.Errorf("覆盖后 = %q, 期望 v2", got)
	}
}

func TestIgnoreErr(t *testing.T) {
	ctx := context.Background()
	//默认为false
	if IsIgnoreErr(ctx) {
		t.Errorf("默认 IsIgnoreErr 应为 false")
	}
	//设为true
	trueCtx := SetIgnoreErr(ctx, true)
	if !IsIgnoreErr(trueCtx) {
		t.Errorf("设为true后 IsIgnoreErr = false")
	}
	//原ctx不受影响（ctx应是不可变的）
	if IsIgnoreErr(ctx) {
		t.Errorf("原ctx被污染")
	}
	//true再设回false
	falseCtx := SetIgnoreErr(trueCtx, false)
	if IsIgnoreErr(falseCtx) {
		t.Errorf("设回false后仍为true")
	}
	//值相同时直接返回原ctx，避免ctx链无谓增长（该实现的有意优化）
	if got := SetIgnoreErr(ctx, false); got != ctx {
		t.Errorf("值相同(false)时应返回原ctx以免ctx链增长")
	}
	if got := SetIgnoreErr(trueCtx, true); got != trueCtx {
		t.Errorf("值相同(true)时应返回原ctx")
	}
}

func TestGenCtx(t *testing.T) {
	ctx := GenCtx()
	if ctx == nil {
		t.Fatalf("GenCtx 返回 nil")
	}
	//必须自带logId
	if GetLogId(ctx) <= 0 {
		t.Errorf("GenCtx 未设置有效 logId: %d", GetLogId(ctx))
	}
	//每次生成的logId应不同
	if GetLogId(GenCtx()) == GetLogId(ctx) {
		t.Errorf("两次 GenCtx 的 logId 相同")
	}
	//新ctx未被取消
	if CtxDone(ctx) {
		t.Errorf("新建 ctx 不应处于 Done 状态")
	}
}

func TestCopyCtx(t *testing.T) {
	old := GenCtx()
	old = SetIgnoreErr(old, true)
	//必须用SetReqId真正写入ctx；GetOrGenReqId只生成不写入，取不到才是预期
	old = SetReqId(old)
	oldLogId := GetLogId(old)
	oldReqId := GetReqId(old)
	if oldReqId <= 0 {
		t.Fatalf("前置条件失败: SetReqId 未写入 reqId")
	}

	//CopyCtx 须复制关键字段
	got := CopyCtx(old)
	if !IsIgnoreErr(got) {
		t.Errorf("CopyCtx 未复制 ignoreErr")
	}
	if GetLogId(got) != oldLogId {
		t.Errorf("CopyCtx logId = %d, 期望 %d", GetLogId(got), oldLogId)
	}
	if GetReqId(got) != oldReqId {
		t.Errorf("CopyCtx reqId = %d, 期望 %d", GetReqId(got), oldReqId)
	}

	//关键用途：脱离原ctx的取消链，父ctx取消不应影响副本
	cancelCtx, cancel := context.WithCancel(old)
	copied := CopyCtx(cancelCtx)
	cancel()
	if !CtxDone(cancelCtx) {
		t.Errorf("原ctx应已取消")
	}
	if CtxDone(copied) {
		t.Errorf("CopyCtx 副本不应随原ctx一起取消（这是该函数的核心用途）")
	}
}

func TestCancelCtx(t *testing.T) {
	//nil 与多个cancel混合传入不能panic
	CancelCtx(nil)
	CancelCtx()

	called := 0
	CancelCtx(func() { called++ }, nil, func() { called++ })
	if called != 2 {
		t.Errorf("CancelCtx 调用次数 = %d, 期望 2（nil应被跳过）", called)
	}

	//真实ctx取消
	ctx, cancel := context.WithCancel(context.Background())
	CancelCtx(cancel)
	if !CtxDone(ctx) {
		t.Errorf("CancelCtx 未取消 ctx")
	}
}

func TestCtxDone(t *testing.T) {
	//未取消
	ctx := context.Background()
	if CtxDone(ctx) {
		t.Errorf("Background ctx 不应为 Done")
	}
	//已取消
	cancelCtx, cancel := context.WithCancel(ctx)
	if CtxDone(cancelCtx) {
		t.Errorf("取消前不应为 Done")
	}
	cancel()
	if !CtxDone(cancelCtx) {
		t.Errorf("取消后应为 Done")
	}
	//CtxDone 必须是非阻塞的：对未取消的ctx也要立即返回
	done := make(chan bool, 1)
	go func() { done <- CtxDone(context.Background()) }()
	select {
	case v := <-done:
		if v {
			t.Errorf("Background = true")
		}
	case <-timeAfterMs(500):
		t.Errorf("CtxDone 阻塞了，应为非阻塞实现")
	}
}
