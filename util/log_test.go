package util

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

func TestLogKeyConstants(t *testing.T) {
	//这些键会写入日志字段并被日志采集依赖，不应被随意改动
	pairs := map[string]string{
		ReqIdKey: "reqid", LogIdKey: "logid", ServerNameKey: "sn",
		IpKey: "ip", CallerKey: "caller",
	}
	for got, want := range pairs {
		if got != want {
			t.Errorf("日志字段常量 = %q, 期望 %q", got, want)
		}
	}
}

func TestGenLogId(t *testing.T) {
	id1 := GenId()
	if id1 <= 0 {
		t.Errorf("GenLogId = %d, 应为正", id1)
	}
	//18位且递增
	if len(Int2Str(id1)) != 18 {
		t.Errorf("GenLogId 位数 = %d", len(Int2Str(id1)))
	}
	if id2 := GenId(); id2 <= id1 {
		t.Errorf("GenLogId 未递增: %d -> %d", id1, id2)
	}
}

func TestGetSetLogId(t *testing.T) {
	//空ctx取不到logId
	if got := GetLogId(context.Background()); got != 0 {
		t.Errorf("空ctx logId = %d, 期望 0", got)
	}
	//设置后可取回
	ctx := SetLogId(context.Background())
	id := GetLogId(ctx)
	if id <= 0 {
		t.Errorf("SetLogId 后 logId = %d", id)
	}
	//幂等：已有logId时不覆盖，保证同一请求链路ID稳定
	again := SetLogId(ctx)
	if GetLogId(again) != id {
		t.Errorf("SetLogId 覆盖了已有logId: %d -> %d", id, GetLogId(again))
	}
	if again != ctx {
		t.Errorf("已有logId时应返回原ctx，避免ctx链增长")
	}
	//字符串形式一致
	if GetLogIdStr(ctx) != Int2Str(id) {
		t.Errorf("GetLogIdStr = %q, 期望 %q", GetLogIdStr(ctx), Int2Str(id))
	}
	//空ctx的字符串形式
	if got := GetLogIdStr(context.Background()); got != "0" {
		t.Errorf(`空ctx GetLogIdStr = %q, 期望 "0"`, got)
	}
}

func TestReSetLogId(t *testing.T) {
	ctx := SetLogId(context.Background())
	old := GetLogId(ctx)
	//与SetLogId不同，ReSetLogId必须强制换新ID
	reset := ReSetLogId(ctx)
	if GetLogId(reset) == old {
		t.Errorf("ReSetLogId 未更换 logId")
	}
	if GetLogId(reset) <= 0 {
		t.Errorf("ReSetLogId 后 logId 非法")
	}
	//原ctx不受影响
	if GetLogId(ctx) != old {
		t.Errorf("原ctx被污染")
	}
}

func TestGenReqId(t *testing.T) {
	//GenReqId 是 SetReqId/GetOrGenReqId 的底层ID来源，此前无任何用例直接覆盖
	id1 := GenId()
	if id1 <= 0 {
		t.Errorf("GenReqId = %d, 应为正整数", id1)
	}
	//与 GenLogId/GenId 同源，均为18位时间序ID
	if got := len(Int2Str(id1)); got != 18 {
		t.Errorf("GenReqId 位数 = %d (%s), 期望 18", got, Int2Str(id1))
	}
	//随时间单调不减；同一时刻可能相等，但绝不能倒退
	for i := 0; i < 50; i++ {
		id2 := GenId()
		if id2 < id1 {
			t.Fatalf("GenReqId 倒退: %d -> %d", id1, id2)
		}
		id1 = id2
	}
	//必须可被 ParseId 解回时间，且与当前时刻接近（证明是时间序ID而非随机数）
	ctx := GenCtx()
	parsed, err := ParseId(ctx, GenId())
	if err != nil {
		t.Fatalf("GenReqId 生成的ID无法解析: %+v", err)
	}
	if diff := time.Since(parsed); diff < -time.Minute || diff > time.Minute {
		t.Errorf("GenReqId 解析出的时间与当前偏差 %v, 过大", diff)
	}
}

func TestReqId(t *testing.T) {
	//空ctx
	if got := GetReqId(context.Background()); got != 0 {
		t.Errorf("空ctx reqId = %d", got)
	}
	//SetReqId 写入
	ctx := SetReqId(context.Background())
	id := GetReqId(ctx)
	if id <= 0 {
		t.Errorf("SetReqId 后 reqId = %d", id)
	}
	//幂等
	if again := SetReqId(ctx); GetReqId(again) != id || again != ctx {
		t.Errorf("SetReqId 非幂等")
	}
	//GetOrGenReqId：有则返回，无则生成但【不写入】ctx
	if got := GetOrGenReqId(ctx); got != id {
		t.Errorf("GetOrGenReqId(已有) = %d, 期望 %d", got, id)
	}
	bare := context.Background()
	gen := GetOrGenReqId(bare)
	if gen <= 0 {
		t.Errorf("GetOrGenReqId(空ctx) = %d, 应生成新ID", gen)
	}
	if GetReqId(bare) != 0 {
		t.Errorf("GetOrGenReqId 不应写入ctx，但ctx中已有reqId")
	}
}

func TestResetAndRmReqId(t *testing.T) {
	ctx := SetReqId(context.Background())
	old := GetReqId(ctx)
	//ReSetReqId 强制换新
	if reset := ReSetReqId(ctx); GetReqId(reset) == old || GetReqId(reset) <= 0 {
		t.Errorf("ReSetReqId = %d, 期望新的正整数", GetReqId(reset))
	}
	//RmReqId 清零，之后GetOrGenReqId应重新生成
	rm := RmReqId(ctx)
	if GetReqId(rm) != 0 {
		t.Errorf("RmReqId 后 reqId = %d, 期望 0", GetReqId(rm))
	}
	if GetOrGenReqId(rm) <= 0 {
		t.Errorf("RmReqId 后 GetOrGenReqId 应能生成新ID")
	}
}

// LogrusHook 是日志字段注入的核心，逐字段验证
func TestParamHookFire(t *testing.T) {
	hook := &LogrusHook{serverName: "test-server"}
	ctx := GenCtx()
	entry := logrus.WithContext(ctx)
	entry.Data = logrus.Fields{}

	if err := hook.Fire(entry); err != nil {
		t.Fatalf("Fire 异常: %+v", err)
	}
	//四个字段必须齐全
	if entry.Data[LogIdKey] != GetLogId(ctx) {
		t.Errorf("logId 字段 = %v, 期望 %d", entry.Data[LogIdKey], GetLogId(ctx))
	}
	if entry.Data[ServerNameKey] != "test-server" {
		t.Errorf("serverName 字段 = %v", entry.Data[ServerNameKey])
	}
	if _, ok := entry.Data[IpKey]; !ok {
		t.Errorf("缺少 ip 字段")
	}
	caller, ok := entry.Data[CallerKey].(string)
	if !ok || caller == "" {
		t.Errorf("caller 字段 = %v", entry.Data[CallerKey])
	}
	//caller 应形如 "文件:行号"，且不能指向logrus内部
	if !strings.Contains(caller, ":") {
		t.Errorf("caller 格式异常: %q", caller)
	}
	if strings.Contains(caller, "sirupsen/logrus") {
		t.Errorf("caller 指向了logrus内部而非业务代码: %q", caller)
	}
	//依赖版本号不应出现在路径中（实现里按@截断）
	if strings.Contains(caller, "@") {
		t.Errorf("caller 未清理依赖版本号: %q", caller)
	}
}

// entry.Context 为nil时不能panic，logId取0
func TestParamHookNilContext(t *testing.T) {
	hook := &LogrusHook{serverName: "s"}
	entry := logrus.NewEntry(logrus.StandardLogger())
	entry.Data = logrus.Fields{}
	if err := hook.Fire(entry); err != nil {
		t.Fatalf("Fire(nil ctx) 异常: %+v", err)
	}
	if entry.Data[LogIdKey] != int64(0) {
		t.Errorf("nil ctx 的 logId = %v, 期望 0", entry.Data[LogIdKey])
	}
}

func TestParamHookLevels(t *testing.T) {
	hook := &LogrusHook{}
	levels := hook.Levels()
	//必须覆盖全部级别，否则部分日志会丢失字段
	if len(levels) != len(logrus.AllLevels) {
		t.Errorf("Levels 数量 = %d, 期望 %d", len(levels), len(logrus.AllLevels))
	}
	has := make(map[logrus.Level]bool)
	for _, l := range levels {
		has[l] = true
	}
	for _, want := range []logrus.Level{logrus.ErrorLevel, logrus.WarnLevel, logrus.InfoLevel, logrus.DebugLevel} {
		if !has[want] {
			t.Errorf("Levels 缺少 %v", want)
		}
	}
}

func TestInitLog(t *testing.T) {
	dir := newTestDir(t)
	chdir(t, dir)
	old := logrus.GetLevel()
	t.Cleanup(func() {
		logrus.SetLevel(old)
		initLog(GetServerName(), "", 1, 100, 30, logrus.InfoLevel)
	})

	initLog("initsvc", "init.log", 1, 2, 3, logrus.DebugLevel)
	if logrus.GetLevel() != logrus.DebugLevel {
		t.Errorf("initLog 未设置级别, got %v", logrus.GetLevel())
	}
	logrus.WithContext(GenCtx()).Debug("测试")
	if info := GetPathInfo(GenCtx(), "log/initsvc/init.log"); info == nil {
		t.Errorf("initLog 日志文件未创建")
	}
	//空参数不panic
	initLog("", "", 1, 1, 1, logrus.InfoLevel)

	//重复初始化不得让钩子累加，否则同一条日志会被Fire多次
	initLog("initsvc", "init.log", 1, 2, 3, logrus.DebugLevel)
	initLog("initsvc", "init.log", 1, 2, 3, logrus.DebugLevel)
	if got := len(logrus.StandardLogger().Hooks[logrus.InfoLevel]); got != 1 {
		t.Errorf("重复initLog后钩子数量 = %d, 期望 1", got)
	}
}

// GinLog 按状态码分三档记录：200=Info、>=500=Error、其余=Warn
func TestGinLog(t *testing.T) {
	cases := []struct {
		status int
		level  string
	}{
		{http.StatusOK, "info"},
		{http.StatusInternalServerError, "error"},
		{http.StatusNotFound, "warning"},
		{http.StatusBadRequest, "warning"},
	}
	for _, c := range cases {
		out := captureLog(t, func() {
			engine := gin.New()
			engine.Use(GinLog)
			engine.GET("/p", func(ctx *gin.Context) { ctx.Status(c.status) })
			w := httptest.NewRecorder()
			engine.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/p?a=1", nil))
			if w.Code != c.status {
				t.Errorf("状态码 = %d, 期望 %d", w.Code, c.status)
			}
		})
		//须记录方法、完整uri（含query）与状态码
		if !strings.Contains(out, "GET") {
			t.Errorf("status=%d 未记录method: %q", c.status, out)
		}
		if !strings.Contains(out, "/p?a=1") {
			t.Errorf("status=%d 未记录完整uri: %q", c.status, out)
		}
		if !strings.Contains(out, fmt.Sprintf("status:%d", c.status)) &&
			!strings.Contains(out, fmt.Sprintf("status=%d", c.status)) {
			t.Errorf("status=%d 未记录状态码: %q", c.status, out)
		}
		//日志级别须与状态码档位匹配
		if !strings.Contains(strings.ToLower(out), c.level[:4]) {
			t.Errorf("status=%d 期望级别 %s, 实际: %q", c.status, c.level, out)
		}
		//须记录耗时
		if !strings.Contains(out, "consume") {
			t.Errorf("status=%d 未记录耗时: %q", c.status, out)
		}
	}
}
