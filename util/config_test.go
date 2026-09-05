package util

import (
	"context"
	"path"
	"sync/atomic"
	"testing"
	"time"

	"github.com/pkg/errors"
)

// fakeConfigHandler 用于在不依赖真实业务配置的前提下驱动 ConfigService
type fakeConfigHandler struct {
	filePath   string
	defaultCfg string
	parsed     atomic.Value //最近一次 ParseConfig 收到的内容
	parseCount atomic.Int64
	//parseErr 用 atomic.Value 存放同一具体类型（*errBox），
	//避免存入不同具体类型导致 atomic.Value 直接panic
	parseErr atomic.Value
}

// errBox 包一层，保证 atomic.Value 中始终是同一具体类型
type errBox struct{ err error }

func newFakeConfigHandler(t *testing.T, defaultCfg string) *fakeConfigHandler {
	t.Helper()
	h := &fakeConfigHandler{filePath: path.Join(newTestDir(t), "conf", "config.yaml"), defaultCfg: defaultCfg}
	h.parsed.Store("")
	h.parseErr.Store(&errBox{})
	return h
}

func (this *fakeConfigHandler) setParseErr(err error) {
	this.parseErr.Store(&errBox{err: err})
}

func (this *fakeConfigHandler) GetPath(ctx context.Context) string   { return this.filePath }
func (this *fakeConfigHandler) GetConfig(ctx context.Context) string { return this.defaultCfg }
func (this *fakeConfigHandler) ParseConfig(ctx context.Context, text string) error {
	this.parseCount.Add(1)
	this.parsed.Store(text)
	if box, ok := this.parseErr.Load().(*errBox); ok && box.err != nil {
		return box.err
	}
	return nil
}
func (this *fakeConfigHandler) lastParsed() string {
	value, _ := this.parsed.Load().(string)
	return value
}

func TestNewConfigService(t *testing.T) {
	h := newFakeConfigHandler(t, "默认配置")
	service := NewConfigService(h)
	if service == nil {
		t.Fatalf("NewConfigService 返回 nil")
	}
	//未加载前配置为空
	ctx := GenCtx()
	if got := service.GetConfig(ctx); got != "" {
		t.Errorf("未加载时 GetConfig = %q, 期望空", got)
	}
}

// 关键回归：首次加载须把默认配置落盘，且写入的是默认内容而非空串
func TestConfigServiceLoadConfigFirstTime(t *testing.T) {
	ctx := GenCtx()
	h := newFakeConfigHandler(t, "默认配置内容")
	service := NewConfigService(h)

	if err := service.LoadConfig(ctx); err != nil {
		t.Fatalf("LoadConfig 异常: %+v", err)
	}
	//配置文件须被创建，且内容为默认配置（曾因用this.text而写出空文件）
	got, err := ReadFile2String(ctx, h.filePath, "")
	if err != nil {
		t.Fatalf("%+v", err)
	}
	if got != "默认配置内容" {
		t.Errorf("落盘内容 = %q, 期望 默认配置内容（写出空文件说明用了未赋值的this.text）", got)
	}
	//内存中的配置也须就绪
	if got = service.GetConfig(ctx); got != "默认配置内容" {
		t.Errorf("GetConfig = %q", got)
	}
	//handler 须被回调解析
	if h.parseCount.Load() != 1 {
		t.Errorf("ParseConfig 调用次数 = %d, 期望 1", h.parseCount.Load())
	}
	if h.lastParsed() != "默认配置内容" {
		t.Errorf("ParseConfig 收到 = %q", h.lastParsed())
	}
}

// 内容未变化时不应重复回调 ParseConfig
func TestConfigServiceLoadConfigNoChange(t *testing.T) {
	ctx := GenCtx()
	h := newFakeConfigHandler(t, "cfg-v1")
	service := NewConfigService(h)

	if err := service.LoadConfig(ctx); err != nil {
		t.Fatalf("%+v", err)
	}
	if h.parseCount.Load() != 1 {
		t.Fatalf("首次 ParseConfig 次数 = %d", h.parseCount.Load())
	}
	//重复加载：内容一致，不应再次解析
	for i := 0; i < 3; i++ {
		if err := service.LoadConfig(ctx); err != nil {
			t.Fatalf("%+v", err)
		}
	}
	if got := h.parseCount.Load(); got != 1 {
		t.Errorf("内容未变仍重复解析 %d 次", got)
	}

	//外部改动文件后，须重新解析
	if err := WriteString2File(ctx, "cfg-v2", h.filePath); err != nil {
		t.Fatalf("%+v", err)
	}
	if err := service.LoadConfig(ctx); err != nil {
		t.Fatalf("%+v", err)
	}
	if got := h.parseCount.Load(); got != 2 {
		t.Errorf("文件变更后解析次数 = %d, 期望 2", got)
	}
	if h.lastParsed() != "cfg-v2" {
		t.Errorf("ParseConfig 收到 = %q, 期望 cfg-v2", h.lastParsed())
	}
	if got := service.GetConfig(ctx); got != "cfg-v2" {
		t.Errorf("GetConfig = %q, 期望 cfg-v2", got)
	}
}

// ParseConfig 报错时不得把配置标记为已生效，否则错误配置会被固化
func TestConfigServiceParseError(t *testing.T) {
	ctx := GenCtx()
	h := newFakeConfigHandler(t, "坏配置")
	h.setParseErr(errors.Errorf("解析失败"))
	service := NewConfigService(h)

	if err := service.LoadConfig(ctx); err == nil {
		t.Errorf("ParseConfig 报错时 LoadConfig 应返回error")
	}
	//解析失败不应更新内存配置
	if got := service.GetConfig(ctx); got != "" {
		t.Errorf("解析失败后 GetConfig = %q, 期望仍为空", got)
	}
	//修复后重新加载须成功
	h.setParseErr(nil)
	if err := service.LoadConfig(ctx); err != nil {
		t.Fatalf("修复后加载异常: %+v", err)
	}
	if got := service.GetConfig(ctx); got != "坏配置" {
		t.Errorf("修复后 GetConfig = %q", got)
	}
}

func TestConfigServiceSaveConfig(t *testing.T) {
	ctx := GenCtx()
	h := newFakeConfigHandler(t, "默认值")
	service := NewConfigService(h)

	//text为空时，SaveConfig 须回落到 handler 的默认配置
	if err := service.SaveConfig(ctx); err != nil {
		t.Fatalf("SaveConfig 异常: %+v", err)
	}
	got, err := ReadFile2String(ctx, h.filePath, "")
	if err != nil {
		t.Fatalf("%+v", err)
	}
	if got != "默认值" {
		t.Errorf("落盘内容 = %q, 期望 默认值", got)
	}

	//SetConfig 后 SaveConfig 须写出新内容
	service.SetConfig(ctx, "手动设置的配置")
	if err = service.SaveConfig(ctx); err != nil {
		t.Fatalf("%+v", err)
	}
	if got, err = ReadFile2String(ctx, h.filePath, ""); err != nil {
		t.Fatalf("%+v", err)
	}
	if got != "手动设置的配置" {
		t.Errorf("落盘内容 = %q, 期望 手动设置的配置", got)
	}
}

func TestConfigServiceSetGetConfig(t *testing.T) {
	ctx := GenCtx()
	service := NewConfigService(newFakeConfigHandler(t, "d"))
	service.SetConfig(ctx, "v1")
	if got := service.GetConfig(ctx); got != "v1" {
		t.Errorf("GetConfig = %q", got)
	}
	service.SetConfig(ctx, "v2")
	if got := service.GetConfig(ctx); got != "v2" {
		t.Errorf("覆盖后 GetConfig = %q", got)
	}
	service.SetConfig(ctx, "")
	if got := service.GetConfig(ctx); got != "" {
		t.Errorf("置空后 GetConfig = %q", got)
	}
}

// Start 须启动守护池并完成首次加载；重复Start应幂等
func TestConfigServiceStart(t *testing.T) {
	ctx := GenCtx()
	h := newFakeConfigHandler(t, "启动配置")
	service := NewConfigService(h)

	if err := service.Start(ctx); err != nil {
		t.Fatalf("Start 异常: %+v", err)
	}
	t.Cleanup(func() { ClosePool(ctx, service.pool) })

	//首次加载须已完成
	if got := service.GetConfig(ctx); got != "启动配置" {
		t.Errorf("Start 后 GetConfig = %q", got)
	}
	if service.pool == nil {
		t.Fatalf("Start 未创建守护池")
	}
	if service.pool.IsClose(ctx) {
		t.Errorf("Start 后守护池不应为关闭态")
	}

	//重复 Start 须幂等，不得重建池
	firstPool := service.pool
	if err := service.Start(ctx); err != nil {
		t.Fatalf("重复 Start 异常: %+v", err)
	}
	if service.pool != firstPool {
		t.Errorf("重复 Start 重建了协程池")
	}

	//关闭后再次 Start 应能重新拉起
	ClosePool(ctx, service.pool)
	if err := service.Start(ctx); err != nil {
		t.Fatalf("关闭后重启异常: %+v", err)
	}
	if service.pool.IsClose(ctx) {
		t.Errorf("重启后池仍为关闭态")
	}
}

// 并发读写配置不得触发 race
func TestConfigServiceConcurrent(t *testing.T) {
	ctx := GenCtx()
	h := newFakeConfigHandler(t, "cfg")
	service := NewConfigService(h)
	if err := service.LoadConfig(ctx); err != nil {
		t.Fatalf("%+v", err)
	}

	done := make(chan bool)
	for i := 0; i < 5; i++ {
		go func() {
			for j := 0; j < 20; j++ {
				service.LoadConfig(ctx)
				service.GetConfig(ctx)
			}
			done <- true
		}()
	}
	for i := 0; i < 5; i++ {
		select {
		case <-done:
		case <-timeAfterMs(20000):
			t.Fatalf("并发加载超时，疑似死锁")
		}
	}
}

// 守护循环使用 ResetLogId 但用 := 遮蔽，不应导致ctx链无限增长
func TestConfigServiceFlushCtxNotLeak(t *testing.T) {
	ctx := GenCtx()
	h := newFakeConfigHandler(t, "cfg")
	service := NewConfigService(h)
	if err := service.Start(ctx); err != nil {
		t.Fatalf("%+v", err)
	}
	t.Cleanup(func() { ClosePool(ctx, service.pool) })

	//跑一小段时间，确认服务稳定且配置可读
	time.Sleep(300 * time.Millisecond)
	if got := service.GetConfig(ctx); got != "cfg" {
		t.Errorf("运行中 GetConfig = %q", got)
	}
	if service.pool.IsClose(ctx) {
		t.Errorf("运行中池被意外关闭")
	}
}
