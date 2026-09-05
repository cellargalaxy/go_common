package util

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/cellargalaxy/go_common/model"
	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
)

func TestGinKeyConstants(t *testing.T) {
	//这些常量参与HTTP协议交互，改动会破坏兼容性
	if AuthorizationKey != "Authorization" {
		t.Errorf("AuthorizationKey = %q", AuthorizationKey)
	}
	if BearerKey != "Bearer" {
		t.Errorf("BearerKey = %q", BearerKey)
	}
	if ClaimsKey != "claims" {
		t.Errorf("ClaimsKey = %q", ClaimsKey)
	}
}

func TestNewHttpResp(t *testing.T) {
	//显式指定各字段
	resp := NewHttpResp(9, "msg", "data")
	if resp.Code != 9 || resp.Msg != "msg" || resp.Data != "data" {
		t.Errorf("NewHttpResp = %+v", resp)
	}
	//nil data 也应保留
	if resp = NewHttpResp(model.SuccessCode, "", nil); resp.Data != nil {
		t.Errorf("nil data = %v", resp.Data)
	}
}

func TestNewHttpRespByMsg(t *testing.T) {
	//空msg视为成功
	resp := NewHttpRespByMsg("d", "")
	if resp.Code != model.SuccessCode {
		t.Errorf("空msg Code = %d, 期望 SuccessCode(%d)", resp.Code, model.SuccessCode)
	}
	if resp.Msg != "" {
		t.Errorf("空msg 时 Msg = %q", resp.Msg)
	}
	if resp.Data != "d" {
		t.Errorf("Data = %v", resp.Data)
	}
	//非空msg视为失败，且data须保留
	resp = NewHttpRespByMsg("d", "出错了")
	if resp.Code != model.FailCode {
		t.Errorf("非空msg Code = %d, 期望 FailCode(%d)", resp.Code, model.FailCode)
	}
	if resp.Msg != "出错了" {
		t.Errorf("Msg = %q", resp.Msg)
	}
	if resp.Data != "d" {
		t.Errorf("失败时 Data 丢失: %v", resp.Data)
	}
}

func TestNewHttpRespByErr(t *testing.T) {
	//nil error 视为成功
	resp := NewHttpRespByErr("d", nil)
	if resp.Code != model.SuccessCode || resp.Msg != "" {
		t.Errorf("nil err = %+v", resp)
	}
	//非nil error：错误信息须落到Msg
	resp = NewHttpRespByErr(nil, errors.Errorf("业务异常"))
	if resp.Code != model.FailCode {
		t.Errorf("Code = %d, 期望 FailCode", resp.Code)
	}
	if !strings.Contains(resp.Msg, "业务异常") {
		t.Errorf("Msg = %q, 应包含错误信息", resp.Msg)
	}
}

func TestGetSetClaims(t *testing.T) {
	ctx := GenCtx()
	//未设置时为nil
	if got := GetClaims(ctx); got != nil {
		t.Errorf("未设置时 GetClaims = %v, 期望 nil", got)
	}
	//设置后可取回同一实例
	claims := &model.Claims{Ip: "1.2.3.4", LogId: 99}
	got := GetClaims(SetClaims(ctx, claims))
	if got == nil {
		t.Fatalf("SetClaims 后取不到")
	}
	if got.Ip != "1.2.3.4" || got.LogId != 99 {
		t.Errorf("claims 内容 = %+v", got)
	}
	//nil claims 不应污染ctx
	if SetClaims(ctx, nil) != ctx {
		t.Errorf("SetClaims(nil) 应返回原ctx")
	}
	if GetClaims(SetClaims(ctx, nil)) != nil {
		t.Errorf("SetClaims(nil) 后应仍取不到claims")
	}
}

func TestPing(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/ping", nil)

	Ping(c)
	if w.Code != http.StatusOK {
		t.Errorf("HTTP状态码 = %d", w.Code)
	}
	//须返回成功码与服务名，且时间戳合理。
	//直接复用 model.PingResponse，避免测试里重复声明json标签而与实现脱节
	var resp struct {
		Code int                `json:"code"`
		Data model.PingResponse `json:"data"`
	}
	if err := JsonString2Struct(w.Body.String(), &resp); err != nil {
		t.Fatalf("响应体解析失败: %+v, body=%s", err, w.Body.String())
	}
	if resp.Code != model.SuccessCode {
		t.Errorf("Code = %d, 期望 SuccessCode", resp.Code)
	}
	if resp.Data.ServerName != GetServerName() {
		t.Errorf("ServerName = %q, 期望 %q", resp.Data.ServerName, GetServerName())
	}
	if resp.Data.Timestamp <= 0 {
		t.Errorf("时间戳 = %d", resp.Data.Timestamp)
	}
	//时间戳应接近当前时间
	if diff := time.Now().Unix() - resp.Data.Timestamp; diff < -5 || diff > 5 {
		t.Errorf("时间戳偏差 %d 秒", diff)
	}
}

func TestSetGinLogId(t *testing.T) {
	//无logId时须生成并写入响应头
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
	setGinLogId(c)

	logId := GetLogId(c)
	if logId <= 0 {
		t.Errorf("未生成 logId: %d", logId)
	}
	if got := w.Header().Get(LogIdKey); got != Int2String(logId) {
		t.Errorf("响应头 logId = %q, 期望 %q", got, Int2String(logId))
	}

	//已有logId时须沿用，保证链路ID贯通
	w = httptest.NewRecorder()
	c, _ = gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
	c.Set(LogIdKey, int64(123456789012345678))
	setGinLogId(c)
	if got := GetLogId(c); got != 123456789012345678 {
		t.Errorf("已有 logId 被覆盖: %d", got)
	}
}

// ClaimsGin 是"尽力解析"中间件：无token或token非法都不阻断请求
func TestClaimsGin(t *testing.T) {
	ctx := GenCtx()
	secret := "s"
	var claims model.Claims
	claims.IssuedAt = time.Now().Unix()
	claims.ExpiresAt = time.Now().Add(time.Hour).Unix()
	claims.Ip = "9.9.9.9"
	claims.LogId = 260904173045123456
	token, err := EnJwt(ctx, secret, claims)
	if err != nil {
		t.Fatalf("%+v", err)
	}

	//Bearer 头
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
	c.Request.Header.Set(AuthorizationKey, "Bearer "+token)
	ClaimsGin(c, secret)
	got := GetClaims(c)
	if got == nil {
		t.Fatalf("Bearer 头未解析出 claims")
	}
	if got.Ip != "9.9.9.9" {
		t.Errorf("claims.Ip = %q", got.Ip)
	}
	//claims 中的 logId 须覆盖到ctx，实现跨服务链路追踪
	if id := GetLogId(c); id != 260904173045123456 {
		t.Errorf("logId 未取自 claims: %d", id)
	}

	//query 参数方式
	w = httptest.NewRecorder()
	c, _ = gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/?"+AuthorizationKey+"="+token, nil)
	ClaimsGin(c, secret)
	if got = GetClaims(c); got == nil || got.Ip != "9.9.9.9" {
		t.Errorf("query 方式未解析出 claims: %v", got)
	}

	//无token：不得阻断，也不应有claims
	w = httptest.NewRecorder()
	c, _ = gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
	ClaimsGin(c, secret)
	if c.IsAborted() {
		t.Errorf("无token时 ClaimsGin 不应中断请求")
	}
	if got = GetClaims(c); got != nil {
		t.Errorf("无token时不应有 claims: %v", got)
	}

	//非法token：不得阻断（这是与 ValidateGin 的关键区别）
	w = httptest.NewRecorder()
	c, _ = gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
	c.Request.Header.Set(AuthorizationKey, "Bearer 非法token")
	ClaimsGin(c, secret)
	if c.IsAborted() {
		t.Errorf("非法token时 ClaimsGin 不应中断请求")
	}

	//非 Bearer 前缀须被忽略
	w = httptest.NewRecorder()
	c, _ = gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
	c.Request.Header.Set(AuthorizationKey, "Basic "+token)
	ClaimsGin(c, secret)
	if got = GetClaims(c); got != nil {
		t.Errorf("非 Bearer 前缀不应被解析: %v", got)
	}
}

// ValidateGin 是鉴权中间件：非法情况必须中断并返回明确原因
func TestValidateGinReject(t *testing.T) {
	ctx := GenCtx()
	secret := "s"

	//无token
	w, c := newGinCtx(http.MethodGet, "/")
	ValidateGin(c, secret)
	assertGinAbort(t, w, c, "无token", model.FailCode, "Authorization非法")

	//错误密钥签发的token
	var claims model.Claims
	claims.IssuedAt = time.Now().Unix()
	claims.ExpiresAt = time.Now().Add(time.Hour).Unix()
	badToken, err := EnJwt(ctx, "wrong-secret", claims)
	if err != nil {
		t.Fatalf("%+v", err)
	}
	w, c = newGinCtx(http.MethodGet, "/")
	c.Request.Header.Set(AuthorizationKey, "Bearer "+badToken)
	ValidateGin(c, secret)
	assertGinAbort(t, w, c, "错误密钥", model.FailCode, "")

	//已过期token
	var expired model.Claims
	expired.IssuedAt = time.Now().Add(-2 * time.Hour).Unix()
	expired.ExpiresAt = time.Now().Add(-time.Hour).Unix()
	expiredToken, err := EnJwt(ctx, secret, expired)
	if err != nil {
		t.Fatalf("%+v", err)
	}
	w, c = newGinCtx(http.MethodGet, "/")
	c.Request.Header.Set(AuthorizationKey, "Bearer "+expiredToken)
	ValidateGin(c, secret)
	assertGinAbort(t, w, c, "过期token", model.FailCode, "")
}

// 合法token须放行
func TestValidateGinPass(t *testing.T) {
	ctx := GenCtx()
	secret := "s"
	var claims model.Claims
	claims.IssuedAt = time.Now().Unix()
	claims.ExpiresAt = time.Now().Add(time.Hour).Unix()
	claims.Ip = "1.1.1.1"
	token, err := EnJwt(ctx, secret, claims)
	if err != nil {
		t.Fatalf("%+v", err)
	}

	_, c := newGinCtx(http.MethodGet, "/")
	c.Request.Header.Set(AuthorizationKey, "Bearer "+token)
	ValidateGin(c, secret)
	if c.IsAborted() {
		t.Errorf("合法token被拒绝")
	}
}

// uri 绑定：claims 指定的uri与实际不符须拒绝，query/fragment 不参与比较
func TestValidateGinUri(t *testing.T) {
	ctx := GenCtx()
	secret := "s"
	newToken := func(uri string) string {
		var claims model.Claims
		claims.IssuedAt = time.Now().Unix()
		claims.ExpiresAt = time.Now().Add(time.Hour).Unix()
		claims.Uri = uri
		token, err := EnJwt(ctx, secret, claims)
		if err != nil {
			t.Fatalf("%+v", err)
		}
		return token
	}

	//uri 匹配：放行
	_, c := newGinCtx(http.MethodGet, "/api/v1/do")
	c.Request.Header.Set(AuthorizationKey, "Bearer "+newToken("/api/v1/do"))
	ValidateGin(c, secret)
	if c.IsAborted() {
		t.Errorf("uri 匹配时被拒绝")
	}

	//带query时仍应匹配（实现会剥掉query）
	_, c = newGinCtx(http.MethodGet, "/api/v1/do?a=1")
	c.Request.Header.Set(AuthorizationKey, "Bearer "+newToken("/api/v1/do"))
	ValidateGin(c, secret)
	if c.IsAborted() {
		t.Errorf("带query时 uri 比较失败")
	}

	//uri 不匹配：须拒绝并返回专用错误码
	w, c := newGinCtx(http.MethodGet, "/api/v1/other")
	c.Request.Header.Set(AuthorizationKey, "Bearer "+newToken("/api/v1/do"))
	ValidateGin(c, secret)
	assertGinAbort(t, w, c, "uri不匹配", model.IllegalUriCode, "请求非法uri")
}

// reqId 防重放：同一reqId第二次必须被拒绝
func TestValidateGinReplay(t *testing.T) {
	ctx := GenCtx()
	secret := "s"
	var claims model.Claims
	claims.IssuedAt = time.Now().Unix()
	claims.ExpiresAt = time.Now().Add(time.Hour).Unix()
	claims.ReqId = GenStringId()
	token, err := EnJwt(ctx, secret, claims)
	if err != nil {
		t.Fatalf("%+v", err)
	}

	//首次放行
	_, c := newGinCtx(http.MethodGet, "/")
	c.Request.Header.Set(AuthorizationKey, "Bearer "+token)
	ValidateGin(c, secret)
	if c.IsAborted() {
		t.Fatalf("首次请求被拒绝")
	}
	//同一token重放：须被拒绝
	w, c := newGinCtx(http.MethodGet, "/")
	c.Request.Header.Set(AuthorizationKey, "Bearer "+token)
	ValidateGin(c, secret)
	assertGinAbort(t, w, c, "重放请求", model.ReRequestCode, "请求非法重放")
}

func TestNewGinGet(t *testing.T) {
	type req struct {
		Name string `form:"name"`
		Age  int    `form:"age"`
	}
	//正常调用：参数须被绑定，返回值须包进HttpResp
	handler := NewGinGet("测试接口", func(ctx context.Context, request req) (any, error) {
		if request.Name != "tom" || request.Age != 18 {
			t.Errorf("参数绑定错误: %+v", request)
		}
		return map[string]string{"ok": "yes"}, nil
	})
	w, c := newGinCtx(http.MethodGet, "/?name=tom&age=18")
	handler(c)
	if w.Code != http.StatusOK {
		t.Errorf("状态码 = %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), `"ok":"yes"`) {
		t.Errorf("响应体 = %s", w.Body.String())
	}

	//service 返回错误：须转成失败响应
	handler = NewGinGet("测试接口", func(ctx context.Context, request req) (any, error) {
		return nil, errors.Errorf("业务失败")
	})
	w, c = newGinCtx(http.MethodGet, "/?name=x")
	handler(c)
	if !strings.Contains(w.Body.String(), "业务失败") {
		t.Errorf("错误未透出: %s", w.Body.String())
	}

	//参数类型非法：须返回解析错误而非panic
	w, c = newGinCtx(http.MethodGet, "/?age=不是数字")
	handler = NewGinGet("测试接口", func(ctx context.Context, request req) (any, error) {
		t.Errorf("参数非法时不应调用 service")
		return nil, nil
	})
	handler(c)
	var resp model.HttpResp
	if err := JsonString2Struct(w.Body.String(), &resp); err != nil {
		t.Fatalf("%+v", err)
	}
	if resp.Code != model.FailCode {
		t.Errorf("参数非法 Code = %d, 期望 FailCode", resp.Code)
	}
}

func TestNewGinPost(t *testing.T) {
	type req struct {
		Name string `json:"name"`
		Age  int    `json:"age"`
	}
	//正常调用
	handler := NewGinPost("测试接口", func(ctx context.Context, request req) (any, error) {
		if request.Name != "tom" || request.Age != 18 {
			t.Errorf("参数绑定错误: %+v", request)
		}
		return "done", nil
	})
	w, c := newGinCtx(http.MethodPost, "/")
	c.Request = httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"name":"tom","age":18}`))
	c.Request.Header.Set("Content-Type", "application/json")
	handler(c)
	if !strings.Contains(w.Body.String(), "done") {
		t.Errorf("响应体 = %s", w.Body.String())
	}

	//非法JSON：须返回解析错误且不调用service
	handler = NewGinPost("测试接口", func(ctx context.Context, request req) (any, error) {
		t.Errorf("JSON非法时不应调用 service")
		return nil, nil
	})
	w, c = newGinCtx(http.MethodPost, "/")
	c.Request = httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{不是json`))
	c.Request.Header.Set("Content-Type", "application/json")
	handler(c)
	var resp model.HttpResp
	if err := JsonString2Struct(w.Body.String(), &resp); err != nil {
		t.Fatalf("%+v", err)
	}
	if resp.Code != model.FailCode {
		t.Errorf("非法JSON Code = %d, 期望 FailCode", resp.Code)
	}
}

// ==== 辅助 ====

func newGinCtx(method, target string) (*httptest.ResponseRecorder, *gin.Context) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(method, target, nil)
	return w, c
}

func assertGinAbort(t *testing.T, w *httptest.ResponseRecorder, c *gin.Context, scene string, wantCode int, wantMsg string) {
	t.Helper()
	if !c.IsAborted() {
		t.Errorf("[%s] 请求未被中断", scene)
	}
	var resp model.HttpResp
	if err := JsonString2Struct(w.Body.String(), &resp); err != nil {
		t.Errorf("[%s] 响应体解析失败: %+v, body=%s", scene, err, w.Body.String())
		return
	}
	if resp.Code != wantCode {
		t.Errorf("[%s] Code = %d, 期望 %d", scene, resp.Code, wantCode)
	}
	if wantMsg != "" && !strings.Contains(resp.Msg, wantMsg) {
		t.Errorf("[%s] Msg = %q, 应包含 %q", scene, resp.Msg, wantMsg)
	}
}
