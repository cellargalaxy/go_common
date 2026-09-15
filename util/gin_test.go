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
	"github.com/golang-jwt/jwt"
	"github.com/pkg/errors"
)

// fakeClaims 自定义claims类型。7b1b8e3 起 ValidateGin 接受任意 Claims 实现，
// 5893d24 起 GetClaims 也泛型化，本类型用于锁定这条链路的端到端行为。
type fakeClaims struct {
	jwt.StandardClaims
	Uri    string `json:"uri,omitempty"`
	LogId  int64  `json:"logid,omitempty"`
	ReqId  int64  `json:"reqid,omitempty"`
	Tenant string `json:"tenant,omitempty"`
}

func (this fakeClaims) GetExpiresAt() int64 { return this.ExpiresAt }
func (this fakeClaims) GetLogId() int64     { return this.LogId }
func (this fakeClaims) GetReqId() int64     { return this.ReqId }
func (this fakeClaims) GetUri() string      { return this.Uri }

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
	if resp = NewHttpResp(http.StatusOK, "", nil); resp.Data != nil {
		t.Errorf("nil data = %v", resp.Data)
	}
}

func TestNewHttpRespByMsg(t *testing.T) {
	//空msg视为成功
	resp := NewHttpRespByMsg("d", "")
	if resp.Code != http.StatusOK {
		t.Errorf("空msg Code = %d, 期望 SuccessCode(%d)", resp.Code, http.StatusOK)
	}
	if resp.Msg != "" {
		t.Errorf("空msg 时 Msg = %q", resp.Msg)
	}
	if resp.Data != "d" {
		t.Errorf("Data = %v", resp.Data)
	}
	//非空msg视为失败，且data须保留
	resp = NewHttpRespByMsg("d", "出错了")
	if resp.Code != http.StatusInternalServerError {
		t.Errorf("非空msg Code = %d, 期望 FailCode(%d)", resp.Code, http.StatusInternalServerError)
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
	if resp.Code != http.StatusOK || resp.Msg != "" {
		t.Errorf("nil err = %+v", resp)
	}
	//非nil error：错误信息须落到Msg
	resp = NewHttpRespByErr(nil, errors.Errorf("业务异常"))
	if resp.Code != http.StatusInternalServerError {
		t.Errorf("Code = %d, 期望 FailCode", resp.Code)
	}
	if !strings.Contains(resp.Msg, "业务异常") {
		t.Errorf("Msg = %q, 应包含错误信息", resp.Msg)
	}
}

func TestGetSetClaims(t *testing.T) {
	ctx := GenCtx()
	//未设置时为nil
	if got := GetClaims[*model.Claims](ctx); got != nil {
		t.Errorf("未设置时 GetClaims = %v, 期望 nil", got)
	}
	//设置后可取回同一实例
	claims := &model.Claims{Ip: "1.2.3.4", LogId: 99}
	got := GetClaims[*model.Claims](SetClaims(ctx, claims))
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
	if GetClaims[*model.Claims](SetClaims(ctx, nil)) != nil {
		t.Errorf("SetClaims(nil) 后应仍取不到claims")
	}
	//空指针形态的nil同样不该污染ctx：117bdb6 泛型化后这条一度失守（详见 IsNil）
	var nilClaims *model.Claims
	if SetClaims(ctx, nilClaims) != ctx {
		t.Errorf("SetClaims(空指针) 应返回原ctx")
	}
	//已有claims的ctx，再塞空值不得把原值顶掉
	had := SetClaims(ctx, claims)
	if got := GetClaims[*model.Claims](SetClaims(had, nilClaims)); got == nil || got.LogId != 99 {
		t.Errorf("塞空值把原claims顶掉了: %v", got)
	}
}

func TestGinPing(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/ping", nil)

	GinPing(c)
	if w.Code != http.StatusOK {
		t.Errorf("HTTP状态码 = %d", w.Code)
	}
	//须返回成功码与服务名，且时间戳合理。
	//直接复用 model.PingData，避免测试里重复声明json标签而与实现脱节
	var resp struct {
		Code int            `json:"code"`
		Data model.PingData `json:"data"`
	}
	if err := JsonStr2Struct(w.Body.String(), &resp); err != nil {
		t.Fatalf("响应体解析失败: %+v, body=%s", err, w.Body.String())
	}
	if resp.Code != http.StatusOK {
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
	setGinLogId(c, GetLogId(c.Request.Context()))

	logId := GetLogId(c.Request.Context())
	if logId <= 0 {
		t.Errorf("未生成 logId: %d", logId)
	}
	if got := w.Header().Get(LogIdKey); got != Int2Str(logId) {
		t.Errorf("响应头 logId = %q, 期望 %q", got, Int2Str(logId))
	}

	//已有logId时须沿用，保证链路ID贯通
	w = httptest.NewRecorder()
	c, _ = gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
	c.Request = c.Request.WithContext(SetCtxValue(c.Request.Context(), LogIdKey, int64(2609041730451234)))
	setGinLogId(c, GetLogId(c.Request.Context()))
	if got := GetLogId(c.Request.Context()); got != 2609041730451234 {
		t.Errorf("已有 logId 被覆盖: %d", got)
	}
	//不得写进 gin.Context.Keys：那个对象请求结束就回 sync.Pool 给下个请求复用
	if _, exist := c.Get(LogIdKey); exist {
		t.Errorf("logId 被写进了 gin.Context.Keys")
	}
}

// ValidateGin 是鉴权中间件：非法情况必须中断并返回明确原因
func TestValidateGinReject(t *testing.T) {
	ctx := GenCtx()
	secret := "s"

	//无token
	w, c := newGinCtx(http.MethodGet, "/")
	ValidateGin(c, secret, &model.Claims{})
	assertGinAbort(t, w, c, "无token", http.StatusUnauthorized, "Authorization非法")

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
	ValidateGin(c, secret, &model.Claims{})
	assertGinAbort(t, w, c, "错误密钥", http.StatusInternalServerError, "")

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
	ValidateGin(c, secret, &model.Claims{})
	assertGinAbort(t, w, c, "过期token", http.StatusInternalServerError, "")
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
	ValidateGin(c, secret, &model.Claims{})
	if c.IsAborted() {
		t.Errorf("合法token被拒绝")
	}
	//放行后须把claims塞进请求ctx，否则下游 GetClaims 恒为nil
	got := GetClaims[*model.Claims](c.Request.Context())
	if got == nil {
		t.Fatalf("放行后取不到 claims")
	}
	if got.Ip != "1.1.1.1" {
		t.Errorf("claims.Ip = %q", got.Ip)
	}

	//query 参数方式同样须能取到token
	_, c = newGinCtx(http.MethodGet, "/?"+AuthorizationKey+"="+token)
	ValidateGin(c, secret, &model.Claims{})
	if c.IsAborted() {
		t.Errorf("query方式的合法token被拒绝")
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
	ValidateGin(c, secret, &model.Claims{})
	if c.IsAborted() {
		t.Errorf("uri 匹配时被拒绝")
	}

	//带query时仍应匹配（实现会剥掉query）
	_, c = newGinCtx(http.MethodGet, "/api/v1/do?a=1")
	c.Request.Header.Set(AuthorizationKey, "Bearer "+newToken("/api/v1/do"))
	ValidateGin(c, secret, &model.Claims{})
	if c.IsAborted() {
		t.Errorf("带query时 uri 比较失败")
	}

	//uri 不匹配：须拒绝并返回专用错误码
	w, c := newGinCtx(http.MethodGet, "/api/v1/other")
	c.Request.Header.Set(AuthorizationKey, "Bearer "+newToken("/api/v1/do"))
	ValidateGin(c, secret, &model.Claims{})
	assertGinAbort(t, w, c, "uri不匹配", http.StatusBadRequest, "请求非法uri")
}

// reqId 防重放：同一reqId第二次必须被拒绝
func TestValidateGinReplay(t *testing.T) {
	ctx := GenCtx()
	secret := "s"
	var claims model.Claims
	claims.IssuedAt = time.Now().Unix()
	claims.ExpiresAt = time.Now().Add(time.Hour).Unix()
	claims.ReqId = GenId()
	token, err := EnJwt(ctx, secret, claims)
	if err != nil {
		t.Fatalf("%+v", err)
	}

	//首次放行
	_, c := newGinCtx(http.MethodGet, "/")
	c.Request.Header.Set(AuthorizationKey, "Bearer "+token)
	ValidateGin(c, secret, &model.Claims{})
	if c.IsAborted() {
		t.Fatalf("首次请求被拒绝")
	}
	//同一token重放：须被拒绝
	w, c := newGinCtx(http.MethodGet, "/")
	c.Request.Header.Set(AuthorizationKey, "Bearer "+token)
	ValidateGin(c, secret, &model.Claims{})
	assertGinAbort(t, w, c, "重放请求", http.StatusConflict, "请求非法重放")
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
	if err := JsonStr2Struct(w.Body.String(), &resp); err != nil {
		t.Fatalf("%+v", err)
	}
	if resp.Code != http.StatusInternalServerError {
		t.Errorf("参数非法 Code = %d, 期望 FailCode", resp.Code)
	}
	//绑定失败也须是"HTTP200 + body业务码"，用BindQuery(MustBindWith)会先写出400再二次写头
	if w.Code != http.StatusOK {
		t.Errorf("参数非法时 HTTP状态码 = %d, 期望 200", w.Code)
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
	if err := JsonStr2Struct(w.Body.String(), &resp); err != nil {
		t.Fatalf("%+v", err)
	}
	if resp.Code != http.StatusInternalServerError {
		t.Errorf("非法JSON Code = %d, 期望 FailCode", resp.Code)
	}
	//同 Get：不得因绑定失败把HTTP状态码改成400
	if w.Code != http.StatusOK {
		t.Errorf("非法JSON时 HTTP状态码 = %d, 期望 200", w.Code)
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
	if err := JsonStr2Struct(w.Body.String(), &resp); err != nil {
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

// 自定义 Claims 实现必须能走通鉴权，并能用 GetClaims 按自身类型取回。
// 7b1b8e3 把 ValidateGin 泛化时 GetClaims 没跟上（固定断言 *model.Claims，
// 自定义claims类型的服务会静默拿到nil），5893d24 把 GetClaims 一并泛型化后闭合。
func TestValidateGinCustomClaims(t *testing.T) {
	ctx := GenCtx()
	secret := "s"
	var claims fakeClaims
	claims.IssuedAt = time.Now().Unix()
	claims.ExpiresAt = time.Now().Add(time.Hour).Unix()
	claims.Tenant = "租户A"
	token, err := EnJwt(ctx, secret, claims)
	if err != nil {
		t.Fatalf("%+v", err)
	}

	_, c := newGinCtx(http.MethodGet, "/")
	c.Request.Header.Set(AuthorizationKey, "Bearer "+token)
	got := &fakeClaims{}
	ValidateGin(c, secret, got)
	if c.IsAborted() {
		t.Fatalf("自定义claims的合法token被拒绝")
	}
	//载荷须被解进调用方传入的对象
	if got.Tenant != "租户A" {
		t.Errorf("自定义claims字段 = %q, 期望 租户A", got.Tenant)
	}
	//GetClaims 按自定义类型取回的必须是同一个实例
	if by := GetClaims[*fakeClaims](c.Request.Context()); by != got {
		t.Errorf("GetClaims[*fakeClaims] 取回 %v, 期望与传入实例相同", by)
	}
	//类型参数给错时返回零值而非panic：GetCtxValue 的断言失败被吞掉(util/ctx.go:9 忽略ok)，
	//故调用方必须自己保证类型参数与 ValidateGin 传入的claims类型一致，否则静默拿到nil
	if by := GetClaims[*model.Claims](c.Request.Context()); by != nil {
		t.Errorf("类型参数不符时应为nil, got %v", by)
	}
}

// claims 必须传指针：三个getter与 jwt.StandardClaims.Valid 都是值接收者，
// 传值(model.Claims{})照样满足 Claims 接口、编译期毫无提示，
// 但 jwt.ParseWithClaims 解不进非指针目标，直接报
// "json: cannot unmarshal object into Go value of type jwt.Claims"，
// 合法请求被全量拒绝。本用例锁定这一行为，防止有人照着值语义写中间件。
func TestValidateGinValueClaims(t *testing.T) {
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

	w, c := newGinCtx(http.MethodGet, "/")
	c.Request.Header.Set(AuthorizationKey, "Bearer "+token)
	ValidateGin(c, secret, model.Claims{}) //故意传值
	assertGinAbort(t, w, c, "claims传值", http.StatusInternalServerError, "JWT解密异常")
}

// 跨服务链路追踪：调用方的logId经 EnDefaultJwt 写进claims(util/codec.go:88 claims.LogId = GetLogId(ctx))，
// 服务端必须取出来覆盖本次请求的logId，两侧日志才能用同一个logId串起来。
//
// 这条链路曾断过：只有 ClaimsGin 做这件事(if claims.LogId > 0 { setGinLogId(c, claims.LogId) })，
// 而它在 681e08d(2026-09-10) 被删除；ValidateGin 从 2023-05-03 的首版起就只做
// setGinLogId(c, GetLogId(c))，从未读过 claims.LogId。
// 于是发送方仍在往JWT里塞logId、接收方却直接丢弃，调用链在服务边界断开。
func TestValidateGinLogIdFromClaims(t *testing.T) {
	ctx := GenCtx()
	secret := "s"
	newToken := func(logId int64) string {
		var claims model.Claims
		claims.IssuedAt = time.Now().Unix()
		claims.ExpiresAt = time.Now().Add(time.Hour).Unix()
		claims.LogId = logId
		token, err := EnJwt(ctx, secret, claims)
		if err != nil {
			t.Fatalf("%+v", err)
		}
		return token
	}

	//前提事实：全新请求的ctx里没有logId，故 setGinLogId(c, GetLogId(c.Request.Context())) 必然现生成一个。
	//本库没有任何中间件会在 ValidateGin 之前写入 LogIdKey
	_, fresh := newGinCtx(http.MethodGet, "/")
	if got := GetLogId(fresh.Request.Context()); got != 0 {
		t.Fatalf("全新请求ctx竟已有logId = %d", got)
	}

	//claims 带logId：必须被采用，且同步到响应头
	const callerLogId = 2609041730451234
	w, c := newGinCtx(http.MethodGet, "/")
	c.Request.Header.Set(AuthorizationKey, "Bearer "+newToken(callerLogId))
	ValidateGin(c, secret, &model.Claims{})
	if c.IsAborted() {
		t.Fatalf("合法token被拒绝")
	}
	if got := GetLogId(c.Request.Context()); got != callerLogId {
		t.Errorf("logId = %d, 期望取自claims的 %d（跨服务链路断开）", got, callerLogId)
	}
	if got := w.Header().Get(LogIdKey); got != Int2Str(callerLogId) {
		t.Errorf("响应头 logId = %q, 期望 %q", got, Int2Str(callerLogId))
	}

	//claims 不带logId：沿用本次请求自己生成的logId，不得被清零
	_, c = newGinCtx(http.MethodGet, "/")
	c.Request.Header.Set(AuthorizationKey, "Bearer "+newToken(0))
	ValidateGin(c, secret, &model.Claims{})
	if got := GetLogId(c.Request.Context()); got <= 0 {
		t.Errorf("claims无logId时应沿用自生成的logId, got %d", got)
	}

	//伪造签名的token不得注入logId：采信发生在验签之后
	var forged model.Claims
	forged.IssuedAt = time.Now().Unix()
	forged.ExpiresAt = time.Now().Add(time.Hour).Unix()
	forged.LogId = callerLogId
	badToken, err := EnJwt(ctx, "wrong-secret", forged)
	if err != nil {
		t.Fatalf("%+v", err)
	}
	_, c = newGinCtx(http.MethodGet, "/")
	c.Request.Header.Set(AuthorizationKey, "Bearer "+badToken)
	ValidateGin(c, secret, &model.Claims{})
	if got := GetLogId(c.Request.Context()); got == callerLogId {
		t.Errorf("验签失败的token注入了logId = %d", got)
	}
}

// 中间件与 service 之间的唯一通路是 c.Request.Context()。
// 曾经 ValidateGin 只把claims写进 gin.Context.Keys(c.Set(ClaimsKey,...))，
// 而 NewGinGet/NewGinPost 交给 service 的是 c.Request.Context()——
// gin.Context.Value 只在 engine.ContextWithFallback 打开时才回落到 Request.Context()，
// 反向(Request.Context() 读 gin.Context.Keys)则永远不通，
// 于是 service 侧 GetClaims 恒为nil、GetLogId 恒为0，logId 在 handler 边界断开。
// 下游实例：jotcash 用 NewGinPost 注册的 5 个路由(jotcash/handler/handler.go:35-39)，
// service 取不到claims → tool.GetToken 报"获取口令，为空" → 加密库连打都打不开。
func TestValidateGinCtxToService(t *testing.T) {
	ctx := GenCtx()
	secret := "s"
	const callerLogId = 2609150011071234
	var claims model.Claims
	claims.IssuedAt = time.Now().Unix()
	claims.ExpiresAt = time.Now().Add(time.Hour).Unix()
	claims.LogId = callerLogId
	claims.Ip = "1.2.3.4"
	token, err := EnJwt(ctx, secret, claims)
	if err != nil {
		t.Fatalf("%+v", err)
	}

	type req struct {
		Name string `form:"name" json:"name"`
	}
	//POST：ValidateGin 放行后，NewGinPost 的 service 必须能拿到同一个claims与同一个logId
	_, c := newGinCtx(http.MethodPost, "/")
	c.Request = httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"name":"tom"}`))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Request.Header.Set(AuthorizationKey, "Bearer "+token)
	got := &model.Claims{}
	ValidateGin(c, secret, got)
	if c.IsAborted() {
		t.Fatalf("合法token被拒绝")
	}
	var postClaims *model.Claims
	var postLogId int64
	NewGinPost("测试接口", func(ctx context.Context, request req) (any, error) {
		postClaims, postLogId = GetClaims[*model.Claims](ctx), GetLogId(ctx)
		return "done", nil
	})(c)
	if postClaims == nil {
		t.Fatalf("NewGinPost 的 service 取不到claims（中间件与service不通）")
	}
	if postClaims != got {
		t.Errorf("service 拿到的claims = %v, 期望与 ValidateGin 传入的是同一实例", postClaims)
	}
	if postLogId != callerLogId {
		t.Errorf("service 侧 logId = %d, 期望 %d（日志链路在handler边界断开）", postLogId, callerLogId)
	}

	//GET：同一条通路，NewGinGet 也须打通
	_, c = newGinCtx(http.MethodGet, "/?name=tom")
	c.Request.Header.Set(AuthorizationKey, "Bearer "+token)
	got = &model.Claims{}
	ValidateGin(c, secret, got)
	if c.IsAborted() {
		t.Fatalf("合法token被拒绝")
	}
	var getClaims *model.Claims
	var getLogId int64
	NewGinGet("测试接口", func(ctx context.Context, request req) (any, error) {
		getClaims, getLogId = GetClaims[*model.Claims](ctx), GetLogId(ctx)
		return "done", nil
	})(c)
	if getClaims != got {
		t.Errorf("NewGinGet 的 service 拿到的claims = %v, 期望与 ValidateGin 传入的是同一实例", getClaims)
	}
	if getLogId != callerLogId {
		t.Errorf("NewGinGet 的 service 侧 logId = %d, 期望 %d", getLogId, callerLogId)
	}

	//未过鉴权时不得凭空出现claims：写入发生在放行那一步
	_, c = newGinCtx(http.MethodPost, "/")
	c.Request = httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"name":"tom"}`))
	c.Request.Header.Set("Content-Type", "application/json")
	ValidateGin(c, secret, &model.Claims{}) //无 Authorization，必被拒
	if !c.IsAborted() {
		t.Fatalf("无token的请求竟被放行")
	}
	if by := GetClaims[*model.Claims](c.Request.Context()); by != nil {
		t.Errorf("鉴权失败却把claims写进了请求ctx: %v", by)
	}
	//鉴权失败也已生成logId（setGinLogId 在入口就跑），须同样对 service 可见
	if by := GetLogId(c.Request.Context()); by <= 0 {
		t.Errorf("请求ctx里的 logId = %d, 期望入口生成的logId", by)
	}
}

// 值一律只写请求ctx，不许写 gin.Context.Keys。
// 原因不是洁癖：gin 在 ServeHTTP 结束时 engine.pool.Put(c)(gin@v1.12.0/gin.go)，
// 下个请求 c.Request = req; c.reset() 把 Keys 直接置nil(gin@v1.12.0/context.go reset)，
// 而 database/sql 的 (*Rows).awaitDone / (*Tx).awaitDone 协程(go1.27 src/database/sql/sql.go:3015/2214)
// 会在请求返回后继续读 ctx.Done()/ctx.Err()——那时 *gin.Context 已经是别人的了。
// 所以本库任何一处都不再把 *gin.Context 当 context.Context 交出去。
func TestValidateGinNotInGinKeys(t *testing.T) {
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

	w, c := newGinCtx(http.MethodGet, "/")
	c.Request.Header.Set(AuthorizationKey, "Bearer "+token)
	got := &model.Claims{}
	ValidateGin(c, secret, got)
	if c.IsAborted() {
		t.Fatalf("合法token被拒绝")
	}
	//请求ctx：claims 与 logId 都在
	if by := GetClaims[*model.Claims](c.Request.Context()); by != got {
		t.Errorf("请求ctx 侧 claims = %v, 期望同一实例", by)
	}
	logId := GetLogId(c.Request.Context())
	if logId <= 0 {
		t.Fatalf("请求ctx 侧 logId = %d", logId)
	}
	//gin.Context.Keys：一个键都不许有
	if len(c.Keys) != 0 {
		t.Errorf("gin.Context.Keys 被写入了 %v", c.Keys)
	}
	if _, exist := c.Get(ClaimsKey); exist {
		t.Errorf("claims 被写进了 gin.Context.Keys")
	}
	if _, exist := c.Get(LogIdKey); exist {
		t.Errorf("logId 被写进了 gin.Context.Keys")
	}
	//响应头照旧带 logId：那是给调用方/人看的，不是进程内通路
	if by := w.Header().Get(LogIdKey); by != Int2Str(logId) {
		t.Errorf("响应头 logId = %q, 期望 %q", by, Int2Str(logId))
	}
}

// 真实 engine 上把三段串起来跑：GinLog(访问日志) → ValidateGin(鉴权) → NewGinPost(业务)。
// 三段现在一律只认 c.Request.Context()，同一个请求打出来的日志必须共用同一个 logId。
// 这条也钉住 GinLog 里 ctx 的取值时机：ValidateGin 是用 c.Request = c.Request.WithContext(...) 写值的，
// 若把 ctx := c.Request.Context() 挪到 c.Next() 之前，访问日志会掉回 logid:0。
func TestGinChainOnlyRequestCtx(t *testing.T) {
	const callerLogId = 2609150011079999
	secret := "s"
	var claims model.Claims
	claims.IssuedAt = time.Now().Unix()
	claims.ExpiresAt = time.Now().Add(time.Hour).Unix()
	claims.LogId = callerLogId
	claims.Ip = "1.1.1.1"
	token, err := EnJwt(GenCtx(), secret, claims)
	if err != nil {
		t.Fatalf("%+v", err)
	}

	type req struct {
		Name string `json:"name"`
	}
	var serviceClaims *model.Claims
	var serviceLogId int64
	var ginKeys map[any]any
	out := captureLog(t, func() {
		engine := gin.New()
		engine.Use(GinLog)
		engine.POST("/p", func(c *gin.Context) {
			ValidateGin(c, secret, &model.Claims{})
			ginKeys = c.Keys //ValidateGin 内部 c.Next() 已把整条链跑完
		}, NewGinPost("测试接口", func(ctx context.Context, request req) (any, error) {
			serviceClaims, serviceLogId = GetClaims[*model.Claims](ctx), GetLogId(ctx)
			return "done", nil
		}))

		w := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodPost, "/p", strings.NewReader(`{"name":"tom"}`))
		request.Header.Set("Content-Type", "application/json")
		request.Header.Set(AuthorizationKey, "Bearer "+token)
		engine.ServeHTTP(w, request)
		if w.Code != http.StatusOK {
			t.Errorf("HTTP状态码 = %d", w.Code)
		}
		if got := w.Header().Get(LogIdKey); got != Int2Str(callerLogId) {
			t.Errorf("响应头 logId = %q, 期望 %q", got, Int2Str(callerLogId))
		}
	})

	if serviceClaims == nil || serviceClaims.Ip != "1.1.1.1" {
		t.Fatalf("service 侧 claims = %v", serviceClaims)
	}
	if serviceLogId != callerLogId {
		t.Errorf("service 侧 logId = %d, 期望 %d", serviceLogId, callerLogId)
	}
	//全程不得往 gin.Context.Keys 里写东西
	if len(ginKeys) != 0 {
		t.Errorf("gin.Context.Keys 被写入了 %v", ginKeys)
	}
	//业务日志与访问日志都得带上同一个 logId
	wantLogId := Int2Str(callerLogId)
	var bizLine, accessLine string
	for _, line := range strings.Split(out, "\n") {
		if strings.Contains(line, "测试接口") {
			bizLine = line
		}
		if strings.Contains(line, "cip") {
			accessLine = line
		}
	}
	if bizLine == "" || !strings.Contains(bizLine, wantLogId) {
		t.Errorf("业务日志未带 logId %s: %q", wantLogId, bizLine)
	}
	if accessLine == "" || !strings.Contains(accessLine, wantLogId) {
		t.Errorf("访问日志未带 logId %s: %q", wantLogId, accessLine)
	}
}
