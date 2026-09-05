package util

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/pkg/errors"
)

type JsonMockResponse struct {
	Id int `json:"id"`
}

func (this *JsonMockResponse) HttpSuccess(ctx context.Context) error {
	if this.Id <= 0 {
		return errors.Errorf("业务校验失败：id 非正数")
	}
	return nil
}

func TestHttpConstants(t *testing.T) {
	//这些默认值影响线上超时与重试行为，改动需谨慎
	if TimeoutDefault != time.Second*3 {
		t.Errorf("TimeoutDefault = %v", TimeoutDefault)
	}
	if SleepDefault != time.Second*3 {
		t.Errorf("SleepDefault = %v", SleepDefault)
	}
	if TryDefault != 3 {
		t.Errorf("TryDefault = %d", TryDefault)
	}
	if UserAgentKey != "User-Agent" {
		t.Errorf("UserAgentKey = %q", UserAgentKey)
	}
	if !strings.Contains(UserAgentDefault, "Mozilla") {
		t.Errorf("UserAgentDefault = %q", UserAgentDefault)
	}
	if len(SpiderSleepDefault) != 3 {
		t.Errorf("SpiderSleepDefault = %v", SpiderSleepDefault)
	}
}

func TestGetSleepTime(t *testing.T) {
	sleeps := []time.Duration{time.Second, 2 * time.Second, 3 * time.Second}
	//按下标取值
	for i, want := range sleeps {
		if got := GetSleepTime(sleeps, i); got != want {
			t.Errorf("GetSleepTime(%d) = %v, 期望 %v", i, got, want)
		}
	}
	//越界取最后一个
	if got := GetSleepTime(sleeps, 99); got != 3*time.Second {
		t.Errorf("越界 = %v, 期望 3s", got)
	}
	//空列表返回1纳秒（非0，避免退避为0导致空转）
	if got := GetSleepTime(nil, 0); got != 1 {
		t.Errorf("空列表 = %v, 期望 1", got)
	}
	if got := GetSleepTime([]time.Duration{}, 5); got != 1 {
		t.Errorf("空切片 = %v, 期望 1", got)
	}
	//命中0值须被抬升为1
	if got := GetSleepTime([]time.Duration{0, time.Second}, 0); got != 1 {
		t.Errorf("0值 = %v, 期望被抬升为 1", got)
	}
	//负数同样被抬升
	if got := GetSleepTime([]time.Duration{-time.Second}, 0); got != 1 {
		t.Errorf("负数 = %v, 期望 1", got)
	}
	//负下标不得panic：CreateHttpClient 的 SetRetryAfter 会传 attempt-1，
	//attempt为0时即为-1，此处按首个退避时间兜底
	if got := GetSleepTime(sleeps, -1); got != time.Second {
		t.Errorf("负下标 = %v, 期望 1s（首个退避时间）", got)
	}
	if got := GetSleepTime(sleeps, -99); got != time.Second {
		t.Errorf("负下标 = %v, 期望 1s", got)
	}
}

func TestGenHttpText(t *testing.T) {
	ctx := GenCtx()
	//仅名称
	if got := genHttpText(ctx, "接口", nil); got != "接口" {
		t.Errorf("genHttpText = %q", got)
	}
	//名称+文案，用中文逗号连接
	if got := genHttpText(ctx, "接口", nil, "异常"); got != "接口，异常" {
		t.Errorf("genHttpText = %q, 期望 接口，异常", got)
	}
	//多段文案
	if got := genHttpText(ctx, "接口", nil, "异常", "重试"); got != "接口，异常，重试" {
		t.Errorf("genHttpText = %q", got)
	}
	//带值时追加冒号
	got := genHttpText(ctx, "接口", 404, "响应码失败")
	if !strings.Contains(got, "404") || !strings.HasPrefix(got, "接口，响应码失败") {
		t.Errorf("genHttpText = %q", got)
	}
	//value为nil不应输出nil字样
	if got = genHttpText(ctx, "接口", nil, "文案"); strings.Contains(got, "nil") {
		t.Errorf("genHttpText 输出了nil: %q", got)
	}
}

func TestDealHttpResponse(t *testing.T) {
	ctx := GenCtx()
	//传入错误须原样报错
	if _, err := DealHttpResponse(ctx, "接口", nil, errors.Errorf("网络错误")); err == nil {
		t.Errorf("传入err时应返回error")
	}
	//响应为空须报错
	if _, err := DealHttpResponse(ctx, "接口", nil, nil); err == nil {
		t.Errorf("响应为空应返回error")
	}

	//200：返回body
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"id":7}`))
	}))
	defer server.Close()
	resp, err := resty.New().R().SetContext(ctx).Get(server.URL)
	body, err := DealHttpResponse(ctx, "接口", resp, err)
	if err != nil {
		t.Fatalf("%+v", err)
	}
	if body != `{"id":7}` {
		t.Errorf("body = %q", body)
	}

	//非200：须报错且错误信息带状态码
	bad := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer bad.Close()
	resp, err = resty.New().R().SetContext(ctx).Get(bad.URL)
	body, err = DealHttpResponse(ctx, "接口", resp, err)
	if err == nil {
		t.Errorf("500 响应应返回error")
	}
	if !strings.Contains(err.Error(), "500") {
		t.Errorf("错误信息未含状态码: %v", err)
	}
	if body != "" {
		t.Errorf("失败时 body 应为空, got %q", body)
	}
}

// HttpApi：成功、JSON非法、业务校验失败三条路径
func TestHttpApi(t *testing.T) {
	ctx := GenCtx()

	//正常
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"id":7}`))
	}))
	defer server.Close()
	var object JsonMockResponse
	if err := HttpApi(ctx, "接口", &object, func() (*resty.Response, error) {
		return GetHttpRequest(ctx).Get(server.URL)
	}); err != nil {
		t.Fatalf("%+v", err)
	}
	if object.Id != 7 {
		t.Errorf("Id = %d, 期望 7", object.Id)
	}

	//响应非法JSON：须报错
	badJson := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`不是json`))
	}))
	defer badJson.Close()
	var object2 JsonMockResponse
	if err := HttpApi(ctx, "接口", &object2, func() (*resty.Response, error) {
		return GetHttpRequest(ctx).Get(badJson.URL)
	}); err == nil {
		t.Errorf("非法JSON应返回error")
	}

	//业务校验失败（HttpSuccess返回错误）：须透出
	zeroId := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"id":0}`))
	}))
	defer zeroId.Close()
	var object3 JsonMockResponse
	err := HttpApi(ctx, "接口", &object3, func() (*resty.Response, error) {
		return GetHttpRequest(ctx).Get(zeroId.URL)
	})
	if err == nil {
		t.Errorf("业务校验失败应返回error")
	}
	if !strings.Contains(err.Error(), "业务校验失败") {
		t.Errorf("未透出业务错误: %v", err)
	}
}

// HttpApiTry：失败须重试到成功；重试次数不少于 len(sleeps)+1
func TestHttpApiTry(t *testing.T) {
	ctx := GenCtx()
	var hits atomic.Int64

	//前两次失败、第三次成功
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if hits.Add(1) < 3 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.Write([]byte(`{"id":7}`))
	}))
	defer server.Close()

	var object JsonMockResponse
	//用极短的退避避免用例变慢
	sleeps := []time.Duration{time.Millisecond, time.Millisecond, time.Millisecond}
	if err := HttpApiTry(ctx, "接口", 0, sleeps, &object, func() (*resty.Response, error) {
		return resty.New().R().SetContext(ctx).Get(server.URL)
	}); err != nil {
		t.Fatalf("重试后应成功: %+v", err)
	}
	if object.Id != 7 {
		t.Errorf("Id = %d", object.Id)
	}
	if got := hits.Load(); got < 3 {
		t.Errorf("实际请求次数 = %d, 期望至少 3（未按预期重试）", got)
	}
}

// 一直失败时须在重试上限后返回错误，而不是无限重试
func TestHttpApiTryExhausted(t *testing.T) {
	ctx := GenCtx()
	var hits atomic.Int64
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	var object JsonMockResponse
	sleeps := []time.Duration{time.Millisecond, time.Millisecond}
	err := HttpApiTry(ctx, "接口", 0, sleeps, &object, func() (*resty.Response, error) {
		return resty.New().R().SetContext(ctx).Get(server.URL)
	})
	if err == nil {
		t.Errorf("持续失败应返回error")
	}
	//try 至少为 len(sleeps)+1 = 3
	if got := hits.Load(); got < 3 {
		t.Errorf("请求次数 = %d, 期望至少 3", got)
	}
	//不应无限重试
	if got := hits.Load(); got > 20 {
		t.Errorf("请求次数 = %d, 疑似无限重试", got)
	}
}

func TestCreateHttpClient(t *testing.T) {
	//基础参数须生效
	client := CreateHttpClient(5*time.Second, 3, []time.Duration{time.Millisecond}, map[string]string{"X-Test": "v"}, true)
	if client == nil {
		t.Fatalf("CreateHttpClient 返回 nil")
	}
	if client.Header.Get("X-Test") != "v" {
		t.Errorf("自定义头未设置: %v", client.Header)
	}
	//未指定UA时须补默认UA，避免被目标站拦截
	if got := client.Header.Get(UserAgentKey); got != UserAgentDefault {
		t.Errorf("默认UA = %q", got)
	}
	//自定义UA须被保留
	custom := CreateHttpClient(time.Second, 0, nil, map[string]string{UserAgentKey: "MyUA"}, false)
	if got := custom.Header.Get(UserAgentKey); got != "MyUA" {
		t.Errorf("自定义UA被覆盖: %q", got)
	}
	//nil header 不应panic
	if got := CreateHttpClient(time.Second, 1, nil, nil, true); got == nil {
		t.Errorf("nil header 时返回 nil")
	}
	//timeout<=0 时不设置超时，也不应panic
	if got := CreateHttpClient(0, 0, nil, nil, true); got == nil {
		t.Errorf("timeout=0 时返回 nil")
	}
}

// 客户端单例：多次获取须为同一实例，避免连接池被反复重建
func TestGetHttpClientSingleton(t *testing.T) {
	c1 := GetHttpClient()
	c2 := GetHttpClient()
	if c1 == nil || c2 == nil {
		t.Fatalf("GetHttpClient 返回 nil")
	}
	if c1 != c2 {
		t.Errorf("GetHttpClient 未复用单例")
	}
	s1 := GetHttpClientSpider()
	s2 := GetHttpClientSpider()
	if s1 != s2 {
		t.Errorf("GetHttpClientSpider 未复用单例")
	}
	//普通客户端与爬虫客户端须相互独立（重试策略不同）
	if c1 == s1 {
		t.Errorf("普通客户端与爬虫客户端为同一实例")
	}
	//Request 每次须新建，且携带传入的ctx
	ctx := GenCtx()
	r1 := GetHttpRequest(ctx)
	r2 := GetHttpRequest(ctx)
	if r1 == nil || r2 == nil {
		t.Fatalf("GetHttpRequest 返回 nil")
	}
	if r1 == r2 {
		t.Errorf("GetHttpRequest 复用了同一Request对象，并发下会互相污染")
	}
	if GetHttpSpiderRequest(ctx) == nil {
		t.Errorf("GetHttpSpiderRequest 返回 nil")
	}
}

// 封禁机制：4xx（非404）会封禁地址，后续请求直接被拦下
func TestHttpBanOnClientRequest(t *testing.T) {
	ctx := GenCtx()
	var hits atomic.Int64
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		w.WriteHeader(http.StatusTooManyRequests) //429，属会触发封禁的4xx
	}))
	defer server.Close()

	client := CreateHttpClient(2*time.Second, 2, []time.Duration{time.Millisecond}, nil, true)
	//首次请求：会真实发出并被判定封禁
	if _, err := client.R().SetContext(ctx).Get(server.URL); err != nil {
		t.Logf("首次请求返回: %v", err)
	}
	first := hits.Load()
	if first < 1 {
		t.Fatalf("首次请求未发出")
	}
	//再次请求同一地址：应在发出前被封禁拦截，服务端不应收到新请求
	_, err := client.R().SetContext(ctx).Get(server.URL)
	if err == nil {
		t.Errorf("被封禁的地址应返回error")
	}
	//必须是 HttpBan 哨兵错误，调用方据此区分"封禁"与普通网络错误。
	//原用例在不匹配时只 t.Logf，任何错误类型都能通过，等于没有校验。
	if !errors.Is(err, HttpBan) {
		t.Errorf("封禁错误应可被 errors.Is(err, HttpBan) 识别, got %v", err)
	}
	if got := hits.Load(); got != first {
		t.Errorf("封禁后仍发出了请求: %d -> %d", first, got)
	}
}

// 404 不触发封禁：后续请求仍可正常发出
func TestHttpNotBanOn404(t *testing.T) {
	ctx := GenCtx()
	var hits atomic.Int64
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	client := CreateHttpClient(2*time.Second, 2, []time.Duration{time.Millisecond}, nil, true)
	client.R().SetContext(ctx).Get(server.URL)
	first := hits.Load()
	client.R().SetContext(ctx).Get(server.URL)
	if got := hits.Load(); got <= first {
		t.Errorf("404 不应触发封禁，但后续请求未发出: %d -> %d", first, got)
	}
}

func TestGetIp(t *testing.T) {
	//GetIp 须安全返回（未取到时为空串），不能panic
	got := GetIp()
	_ = got
	//写入后可读回，验证原子变量读写自洽
	old := ip.Load()
	ip.Store("1.2.3.4")
	if got = GetIp(); got != "1.2.3.4" {
		t.Errorf("GetIp = %q, 期望 1.2.3.4", got)
	}
	//恢复原值，避免影响其他用例的日志字段
	if old != nil {
		ip.Store(old)
	} else {
		ip.Store("")
	}
}

func TestParseCurl(t *testing.T) {
	ctx := GenCtx()
	curl := `curl 'https://example.com/api?a=1' \
  -H 'Content-Type: application/json' \
  -H 'Cookie: k=v' \
  --data-raw '{"key":"value"}'`

	param, err := ParseCurl(ctx, curl)
	if err != nil {
		t.Fatalf("%+v", err)
	}
	if param.Url != "https://example.com/api?a=1" {
		t.Errorf("Url = %q", param.Url)
	}
	if param.Header["Content-Type"] != "application/json" {
		t.Errorf("Content-Type = %q", param.Header["Content-Type"])
	}
	if param.Header["Cookie"] != "k=v" {
		t.Errorf("Cookie = %q", param.Header["Cookie"])
	}
	if param.Body != `{"key":"value"}` {
		t.Errorf("Body = %q", param.Body)
	}

	//双引号形式同样须支持
	param, err = ParseCurl(ctx, "curl \"https://example.com\"\n  -H \"A: b\"")
	if err != nil {
		t.Fatalf("%+v", err)
	}
	if param.Url != "https://example.com" {
		t.Errorf("双引号 Url = %q", param.Url)
	}
	if param.Header["A"] != "b" {
		t.Errorf("双引号 Header = %v", param.Header)
	}

	//空输入：不能panic，Header须已初始化可直接写
	param, err = ParseCurl(ctx, "")
	if err != nil {
		t.Fatalf("%+v", err)
	}
	if param == nil {
		t.Fatalf("空输入返回 nil")
	}
	if param.Header == nil {
		t.Errorf("Header 未初始化，调用方写入会panic")
	}
	if param.Url != "" {
		t.Errorf("空输入 Url = %q", param.Url)
	}
	//无冒号的-H须被跳过而非panic
	if param, err = ParseCurl(ctx, "curl 'https://x.com'\n  -H 'NoColonHeader'"); err != nil {
		t.Fatalf("%+v", err)
	}
	if len(param.Header) != 0 {
		t.Errorf("非法头应被跳过: %v", param.Header)
	}
}

// ExecCurl 的参数校验分支，不依赖外网
func TestExecCurlValidate(t *testing.T) {
	ctx := GenCtx()
	//非法方法须被拒绝
	for _, method := range []string{"PUT", "DELETE", "PATCH", "非法"} {
		if _, err := ExecCurl(ctx, "接口", method, "http://example.com", nil); err == nil {
			t.Errorf("方法 %q 应被拒绝", method)
		}
	}
	//空url须被拒绝
	for _, url := range []string{"", "   "} {
		if _, err := ExecCurl(ctx, "接口", "GET", url, nil); err == nil {
			t.Errorf("空url应被拒绝")
		}
	}
	//方法大小写不敏感：小写get应被接受（此处用不可达地址，只验证未因方法被拒）
	_, err := ExecCurl(ctx, "接口", "get", "http://127.0.0.1:1/nothing", nil)
	if err != nil && strings.Contains(err.Error(), "非法方法") {
		t.Errorf("小写方法应被接受: %v", err)
	}
}

// ExecCurl 打通本地httptest服务，验证真实取数与状态码解析
func TestExecCurlLocal(t *testing.T) {
	if _, _, err := ExecCommand(GenCtx(), "command -v curl"); err != nil {
		t.Skip("环境无curl命令，跳过")
	}
	ctx := GenCtx()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get(UserAgentKey) == "" {
			t.Errorf("请求未携带UA")
		}
		w.Write([]byte(`{"page":1}`))
	}))
	defer server.Close()

	data, err := ExecCurl(ctx, "接口", "", server.URL, nil)
	if err != nil {
		t.Fatalf("ExecCurl 异常: %+v", err)
	}
	if !strings.Contains(data, `"page":1`) {
		t.Errorf("响应体 = %q", data)
	}

	//非200须报错并带状态码
	bad := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer bad.Close()
	if _, err = ExecCurl(ctx, "接口", "GET", bad.URL, nil); err == nil {
		t.Errorf("500 响应应返回error")
	}
}
