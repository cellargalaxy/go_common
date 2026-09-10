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
}

func TestDealHttpClientResp(t *testing.T) {
	ctx := GenCtx()
	//传入错误须原样报错
	if _, err := DealHttpClientResp(ctx, "接口", nil, errors.Errorf("网络错误")); err == nil {
		t.Errorf("传入err时应返回error")
	}
	//响应为空须报错
	if _, err := DealHttpClientResp(ctx, "接口", nil, nil); err == nil {
		t.Errorf("响应为空应返回error")
	}

	//200：返回body
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"id":7}`))
	}))
	defer server.Close()
	resp, err := resty.New().R().SetContext(ctx).Get(server.URL)
	body, err := DealHttpClientResp(ctx, "接口", resp, err)
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
	body, err = DealHttpClientResp(ctx, "接口", resp, err)
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

func TestCreateHttpClient(t *testing.T) {
	//基础参数须生效
	client := CreateHttpClient(5*time.Second, []time.Duration{time.Millisecond}, map[string]string{"X-Test": "v"}, true)
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
	custom := CreateHttpClient(time.Second, nil, map[string]string{UserAgentKey: "MyUA"}, false)
	if got := custom.Header.Get(UserAgentKey); got != "MyUA" {
		t.Errorf("自定义UA被覆盖: %q", got)
	}
	//nil header 不应panic
	if got := CreateHttpClient(time.Second, nil, nil, true); got == nil {
		t.Errorf("nil header 时返回 nil")
	}
	//timeout<=0 时不设置超时，也不应panic
	if got := CreateHttpClient(0, nil, nil, true); got == nil {
		t.Errorf("timeout=0 时返回 nil")
	}
}

func TestHttpBanOnClientRequest(t *testing.T) {
	ctx := GenCtx()
	var hits atomic.Int64
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		w.WriteHeader(http.StatusTooManyRequests) //429，属会触发封禁的4xx
	}))
	defer server.Close()

	client := CreateHttpClient(2*time.Second, []time.Duration{time.Millisecond, time.Millisecond}, nil, true)
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
	//须是封禁错误，调用方据此区分"封禁"与普通网络错误。
	//哨兵错误现为 CreateHttpClient 内的闭包变量，包外不可见，只能按文案断言
	if !strings.Contains(err.Error(), "HTTP请求封禁") {
		t.Errorf("封禁错误文案 = %v, 期望含 HTTP请求封禁", err)
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

	client := CreateHttpClient(2*time.Second, []time.Duration{time.Millisecond, time.Millisecond}, nil, true)
	client.R().SetContext(ctx).Get(server.URL)
	first := hits.Load()
	client.R().SetContext(ctx).Get(server.URL)
	if got := hits.Load(); got <= first {
		t.Errorf("404 不应触发封禁，但后续请求未发出: %d -> %d", first, got)
	}
}

func TestGetIP(t *testing.T) {
	//GetIP 须安全返回（未取到时为空串），不能panic
	got := GetIP()
	_ = got
	//写入后可读回，验证原子变量读写自洽
	old := ip.Load()
	ip.Store("1.2.3.4")
	if got = GetIP(); got != "1.2.3.4" {
		t.Errorf("GetIP = %q, 期望 1.2.3.4", got)
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

	url, header, body, err := ParseCurl(ctx, curl)
	if err != nil {
		t.Fatalf("%+v", err)
	}
	if url != "https://example.com/api?a=1" {
		t.Errorf("Url = %q", url)
	}
	if header["Content-Type"] != "application/json" {
		t.Errorf("Content-Type = %q", header["Content-Type"])
	}
	if header["Cookie"] != "k=v" {
		t.Errorf("Cookie = %q", header["Cookie"])
	}
	if body != `{"key":"value"}` {
		t.Errorf("Body = %q", body)
	}

	//双引号形式同样须支持
	url, header, _, err = ParseCurl(ctx, "curl \"https://example.com\"\n  -H \"A: b\"")
	if err != nil {
		t.Fatalf("%+v", err)
	}
	if url != "https://example.com" {
		t.Errorf("双引号 Url = %q", url)
	}
	if header["A"] != "b" {
		t.Errorf("双引号 Header = %v", header)
	}

	//空输入：不能panic，Header须已初始化可直接写
	url, header, body, err = ParseCurl(ctx, "")
	if err != nil {
		t.Fatalf("%+v", err)
	}
	if header == nil {
		t.Errorf("Header 未初始化，调用方写入会panic")
	}
	if url != "" || body != "" {
		t.Errorf("空输入 Url = %q, Body = %q", url, body)
	}
	//无冒号的-H须被跳过而非panic
	if _, header, _, err = ParseCurl(ctx, "curl 'https://x.com'\n  -H 'NoColonHeader'"); err != nil {
		t.Fatalf("%+v", err)
	}
	if len(header) != 0 {
		t.Errorf("非法头应被跳过: %v", header)
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
	//curl 会继承环境代理，本地httptest若被代理拦截会拿到无关状态码
	t.Setenv("http_proxy", "")
	t.Setenv("https_proxy", "")
	t.Setenv("no_proxy", "127.0.0.1,localhost")
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

// LoadIP / flushIP：此前无任何用例直接调用。
// LoadIP 依赖外网端点，不能让用例的成败取决于网络；这里只锁定"必须安全返回、
// 不panic、返回值可直接交给 ip.Store 使用"这类与网络无关的契约。
func TestLoadIP(t *testing.T) {
	ctx := GenCtx()

	//无论外网是否可达，都不得panic，且返回值必须是可安全使用的字符串
	got := LoadIP(ctx)
	if strings.ContainsAny(got, "\x00") {
		t.Errorf("LoadIP 返回含NUL字节: %q", got)
	}
	//取到内容时，应是去除首尾空白后仍非空的一行文本（flushIP 依赖这一点做判空）
	if trimmed := strings.TrimSpace(got); trimmed != "" {
		if strings.Contains(trimmed, "\n") {
			t.Errorf("LoadIP 返回多行内容 %q, flushIP 会把整段存进ip字段", trimmed)
		}
	} else {
		t.Logf("LoadIP 未取到内容（外网不可达时属正常）: %q", got)
	}

	//ctx 已取消时必须快速返回且不panic，不能挂住调用方
	cancelled, cancel := context.WithCancel(ctx)
	cancel()
	start := time.Now()
	_ = LoadIP(cancelled)
	if elapsed := time.Since(start); elapsed > 5*time.Second {
		t.Errorf("已取消ctx下 LoadIP 耗时 %v, 未及时返回", elapsed)
	}
}

// flushIP 是守护任务体：只校验它对空返回值的保护逻辑，
// 即"取到空串时不得把 ip 覆盖成空"，这是该函数唯一与网络无关的关键分支。
func TestFlushLoadIPKeepsOldOnEmpty(t *testing.T) {
	ctx := GenCtx()
	old := ip.Load()
	t.Cleanup(func() {
		if old != nil {
			ip.Store(old)
		} else {
			ip.Store("")
		}
	})

	//预置一个已知IP，模拟此前已成功获取过
	ip.Store("9.9.9.9")

	//用已取消的ctx驱动一次守护体：内部取IP大概率为空且循环会立刻退出，
	//关键断言是——即使这一轮没取到，也不能把已有的IP擦成空串
	cancelled, cancel := context.WithCancel(ctx)
	cancel()
	done := make(chan bool, 1)
	go func() {
		flushIP(cancelled, nil)
		done <- true
	}()
	select {
	case <-done:
	case <-timeAfterMs(15000):
		t.Fatalf("flushIP 在已取消ctx下未退出")
	}

	if got := GetIP(); got == "" {
		t.Errorf("flushIP 把已有IP擦成了空串，GetIP = %q", got)
	}
}
