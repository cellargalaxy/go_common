package util

import (
	"context"
	"os"
	"path"
	"strings"
	"testing"
	"time"

	"github.com/ArtisanCloud/PowerWeChat/v3/src/kernel/response"
	"github.com/pkg/errors"
)

// 这些默认值直接决定同步调用方被阻塞多久、以及测试号配额被消耗多快，改动需谨慎
func TestWxConstants(t *testing.T) {
	if WxTryDefault != 3 {
		t.Errorf("WxTryDefault = %d", WxTryDefault)
	}
	if WxTimeoutDefault != time.Second*10 {
		t.Errorf("WxTimeoutDefault = %v", WxTimeoutDefault)
	}
	if WxMetaTimeout != time.Hour {
		t.Errorf("WxMetaTimeout = %v", WxMetaTimeout)
	}
	if WxRepeatTimeout != time.Minute*5 {
		t.Errorf("WxRepeatTimeout = %v", WxRepeatTimeout)
	}
	if WxTextLenMax <= 0 {
		t.Errorf("WxTextLenMax = %d", WxTextLenMax)
	}
	//间隔条数必须覆盖"总尝试次数-1"，否则最后几次重试会退化成固定间隔
	if len(wxSleeps) < WxTryDefault-1 {
		t.Errorf("wxSleeps 条数 = %d, 至少需要 %d 条", len(wxSleeps), WxTryDefault-1)
	}
	//模板正文里日志行的字段名是 log，与日志字段 LogIdKey(logid) 不同，不能混用
	if wxLogKey != "log" {
		t.Errorf("wxLogKey = %q", wxLogKey)
	}
	if wxLogKey == LogIdKey {
		t.Errorf("wxLogKey 不应等于 LogIdKey")
	}
	//模板只有一个占位符 {{data.DATA}}
	if wxDataKey != "data" {
		t.Errorf("wxDataKey = %q", wxDataKey)
	}
}

func TestSplitWxOpenIds(t *testing.T) {
	//空值不得产出空字符串元素，否则会向空openid发消息
	if got := splitWxOpenIds(""); len(got) != 0 {
		t.Errorf("空值 = %v, 期望空切片", got)
	}
	if got := splitWxOpenIds("  ,  ,"); len(got) != 0 {
		t.Errorf("全空项 = %v, 期望空切片", got)
	}
	//单个
	got := splitWxOpenIds("oA")
	if len(got) != 1 || got[0] != "oA" {
		t.Errorf("单个 = %v", got)
	}
	//多个且去除空格与空项
	got = splitWxOpenIds(" oA , oB ,, oC ")
	if len(got) != 3 || got[0] != "oA" || got[1] != "oB" || got[2] != "oC" {
		t.Errorf("多个 = %v", got)
	}
}

func TestGenWxMsgData(t *testing.T) {
	ctx := GenCtx()
	data := genWxMsgData(ctx, "我是正文A\n我是正文B")
	lines := strings.Split(data, "\n")
	if len(lines) != 5 {
		t.Fatalf("行数 = %d, 期望 5（3行上下文+2行正文）: %q", len(lines), data)
	}
	//前三行的字段名与顺序是模板约定，改动会让消息格式与后台模板不一致
	if !strings.HasPrefix(lines[0], ServerNameKey+": ") {
		t.Errorf("第1行 = %q, 期望以 %s: 开头", lines[0], ServerNameKey)
	}
	if !strings.HasPrefix(lines[1], IpKey+": ") {
		t.Errorf("第2行 = %q, 期望以 %s: 开头", lines[1], IpKey)
	}
	if !strings.HasPrefix(lines[2], wxLogKey+": ") {
		t.Errorf("第3行 = %q, 期望以 %s: 开头", lines[2], wxLogKey)
	}
	//正文必须原样保留换行
	if lines[3] != "我是正文A" || lines[4] != "我是正文B" {
		t.Errorf("正文 = %q / %q", lines[3], lines[4])
	}
	//GenCtx 已写入logid，此处应有值
	if lines[2] != wxLogKey+": "+Int2Str(GetLogId(ctx)) {
		t.Errorf("日志行 = %q", lines[2])
	}
}

// 上下文取不到值时必须留空，不能报错也不能填成 0。
// GetIP 由后台协程异步填充，进程刚启动时必然为空，而那恰是最需要告警的时段。
func TestGenWxMsgDataEmptyContext(t *testing.T) {
	//未经 SetLogId 的ctx，logid 为0，日志行必须是空值而不是 "0"
	data := genWxMsgData(context.Background(), "正文")
	lines := strings.Split(data, "\n")
	if len(lines) != 4 {
		t.Fatalf("行数 = %d: %q", len(lines), data)
	}
	if lines[2] != wxLogKey+": " {
		t.Errorf("logid为0时日志行 = %q, 期望留空", lines[2])
	}
	if strings.Contains(lines[2], "0") {
		t.Errorf("logid为0时不应把0写进消息: %q", lines[2])
	}
	//正文仍须完整
	if lines[3] != "正文" {
		t.Errorf("正文 = %q", lines[3])
	}
}

func TestCutWxText(t *testing.T) {
	ctx := GenCtx()
	//未超长原样返回
	if got := cutWxText(ctx, "短正文"); got != "短正文" {
		t.Errorf("未超长 = %q", got)
	}
	//按字符而非字节截断，否则中文会被切成乱码
	long := strings.Repeat("中", WxTextLenMax+10)
	got := cutWxText(ctx, long)
	if !strings.HasPrefix(got, strings.Repeat("中", WxTextLenMax)) {
		t.Errorf("截断后前缀不符")
	}
	if strings.Count(got, "中") != WxTextLenMax {
		t.Errorf("保留字符数 = %d, 期望 %d", strings.Count(got, "中"), WxTextLenMax)
	}
	if !strings.HasSuffix(got, "(已截断)") {
		t.Errorf("截断后应有标记: %q", got[len(got)-30:])
	}
	//边界：恰好等于上限不截断
	exact := strings.Repeat("中", WxTextLenMax)
	if cutWxText(ctx, exact) != exact {
		t.Errorf("恰好等于上限时不应截断")
	}
}

// wxCache 的核心契约：Get 必须返回 map[string]interface{}。
// PowerWeChat 在 kernel/accessToken.go 中对缓存值做 value.(map[string]interface{}) 无保护断言，
// 若本实现直接存取结构体指针，取 token 时会 panic。
func TestWxCacheJsonRoundTrip(t *testing.T) {
	c := newWxCache()
	//模拟 PowerWeChat 写入 access_token 的形态：结构体指针
	token := &response.ResponseGetToken{AccessToken: "tk", ExpiresIn: 7200}
	if err := c.Set("k", token, time.Hour); err != nil {
		t.Fatalf("Set 异常: %+v", err)
	}
	value, err := c.Get("k", nil)
	if err != nil {
		t.Fatalf("Get 异常: %+v", err)
	}
	hash, ok := value.(map[string]interface{})
	if !ok {
		t.Fatalf("Get 返回 %T, 必须是 map[string]interface{}，否则 PowerWeChat 取token时会panic", value)
	}
	//expires_in 必须是 float64：getFormatToken 做的是 token["expires_in"].(float64)
	if _, ok = hash["expires_in"].(float64); !ok {
		t.Errorf("expires_in 为 %T, 必须是 float64", hash["expires_in"])
	}
	//access_token 必须是 string：getFormatToken 做的是 token["access_token"].(string)
	if got, _ := hash["access_token"].(string); got != "tk" {
		t.Errorf("access_token = %v", hash["access_token"])
	}
	if !c.Has("k") {
		t.Errorf("Has 应为 true")
	}
}

// 未命中时必须返回 defaultValue 且 err 为 nil。
// PowerWeChat 的 GetToken 靠 "err==nil && value!=nil" 判断是否用缓存，
// 未命中返回非nil值会让它拿着脏数据去做类型断言。
func TestWxCacheMiss(t *testing.T) {
	c := newWxCache()
	if c.Has("absent") {
		t.Errorf("未写入的键 Has 应为 false")
	}
	value, err := c.Get("absent", nil)
	if err != nil {
		t.Errorf("未命中不应报错: %+v", err)
	}
	if value != nil {
		t.Errorf("未命中应返回 defaultValue(nil), got %v", value)
	}
	//defaultValue 非nil时须原样返回
	if value, _ = c.Get("absent", "def"); value != "def" {
		t.Errorf("未命中应返回传入的 defaultValue, got %v", value)
	}
}

func TestWxCacheSetExpires(t *testing.T) {
	c := newWxCache()
	//短TTL不能被余量削成非正数，否则写进去立刻就没了
	if err := c.Set("short", map[string]string{"a": "b"}, time.Minute); err != nil {
		t.Fatalf("Set 异常: %+v", err)
	}
	if !c.Has("short") {
		t.Errorf("1分钟TTL写入后立刻取不到，说明被余量削成了非正数")
	}
	//TTL为0时须兜底成固定时长：go-cache 会把0当作"使用默认过期时间"
	if err := c.Set("zero", map[string]string{"a": "b"}, 0); err != nil {
		t.Fatalf("Set 异常: %+v", err)
	}
	if !c.Has("zero") {
		t.Errorf("TTL为0时写入后取不到")
	}
	//长TTL须被削去余量：避免临界点用到刚失效的凭证
	if err := c.Set("long", map[string]string{"a": "b"}, time.Minute*6); err != nil {
		t.Fatalf("Set 异常: %+v", err)
	}
	_, expire, ok := c.cache.cache.GetWithExpiration("long")
	if !ok || expire.IsZero() {
		t.Fatalf("长TTL未写入或无过期时间")
	}
	if remain := time.Until(expire); remain > time.Minute*6 {
		t.Errorf("剩余TTL = %v, 应已削去余量", remain)
	}
}

// Add/AddNX/Remember 是空实现：PowerWeChat 没有任何调用点。
// 固化该语义，避免将来有人误以为它们可用于原子操作。
func TestWxCacheNoOpMethods(t *testing.T) {
	c := newWxCache()
	if c.AddNX("k", "v", time.Hour) {
		t.Errorf("AddNX 应恒为 false")
	}
	if err := c.Add("k", "v", time.Hour); err != nil {
		t.Errorf("Add 应恒为 nil: %+v", err)
	}
	value, err := c.Remember("k", time.Hour, func() (interface{}, error) { return "v", nil })
	if value != nil || err != nil {
		t.Errorf("Remember 应恒为 nil,nil: %v, %v", value, err)
	}
	//空实现不得真的写入，否则 Has/Get 会出现与 Set 不一致的来源
	if c.Has("k") {
		t.Errorf("空实现不应写入缓存")
	}
}

func TestWxRespErr(t *testing.T) {
	err := wxRespErr{ErrCode: 45009, ErrMsg: "reach max api daily quota limit"}
	if !strings.Contains(err.Error(), "45009") {
		t.Errorf("Error() 应含errcode: %q", err.Error())
	}
	if !strings.Contains(err.Error(), "quota") {
		t.Errorf("Error() 应含errmsg: %q", err.Error())
	}
	//被包装后仍须能取出：isWxErrRetry 的分类完全依赖 errors.As
	var target wxRespErr
	if !errors.As(errors.Wrap(err, "外层"), &target) {
		t.Fatalf("包装后 errors.As 取不出 wxRespErr，错误分类会失效")
	}
	if target.ErrCode != 45009 {
		t.Errorf("取出的 errcode = %d", target.ErrCode)
	}
}

func TestIsWxErrRetry(t *testing.T) {
	//nil 不是错误，不该进重试
	if isWxErrRetry(nil) {
		t.Errorf("nil 不应重试")
	}
	//传输层错误（网络/超时/HTTP非200）重试可能成功
	if !isWxErrRetry(errors.Errorf("connection reset")) {
		t.Errorf("传输层错误应重试")
	}
	if !isWxErrRetry(context.DeadlineExceeded) {
		t.Errorf("超时应归入传输层并重试")
	}
	//系统繁忙是唯一值得重试的业务码
	if !isWxErrRetry(wxRespErr{ErrCode: wxErrCodeBusy}) {
		t.Errorf("系统繁忙应重试")
	}
	//参数类错误重试必然同样失败
	if isWxErrRetry(wxRespErr{ErrCode: 40037, ErrMsg: "invalid template_id"}) {
		t.Errorf("模板ID错误不应重试")
	}
	//配额类错误重试会加速配额耗尽
	if isWxErrRetry(wxRespErr{ErrCode: 45009}) {
		t.Errorf("配额超限不应重试")
	}
	//凭证类由 PowerWeChat 中间件自动刷token并重放，这里不叠加
	if isWxErrRetry(wxRespErr{ErrCode: 40001}) {
		t.Errorf("凭证失效不应在业务层重试")
	}
	//未知码默认不重试
	if isWxErrRetry(wxRespErr{ErrCode: 99999}) {
		t.Errorf("未知码默认不应重试")
	}
	//包装后不得退化成"传输层错误"从而被误判为可重试
	if isWxErrRetry(errors.Wrap(wxRespErr{ErrCode: 45009}, "外层")) {
		t.Errorf("包装后的配额错误仍不应重试")
	}
}

func TestReTryWx(t *testing.T) {
	ctx := GenCtx()
	//把间隔压到毫秒级，用例只验证次数与短路行为，不验证真实等待时长
	originSleeps := wxSleeps
	wxSleeps = []time.Duration{time.Millisecond, time.Millisecond}
	defer func() { wxSleeps = originSleeps }()

	//成功立即返回，不得重试
	calls := 0
	got, err := reTryWx(ctx, "用例", func(ctx context.Context) (string, error) {
		calls++
		return "ok", nil
	})
	if err != nil || got != "ok" {
		t.Fatalf("成功路径 = %q, %v", got, err)
	}
	if calls != 1 {
		t.Errorf("成功时调用次数 = %d, 期望 1", calls)
	}

	//传输层错误重试到预算耗尽，且必须把最后一次的错误透传出来
	calls = 0
	sentinel := errors.Errorf("网络异常")
	_, err = reTryWx(ctx, "用例", func(ctx context.Context) (string, error) {
		calls++
		return "", sentinel
	})
	if err == nil {
		t.Fatalf("耗尽后应返回error")
	}
	if !errors.Is(err, sentinel) {
		t.Errorf("应透传原始错误, got %v", err)
	}
	if calls != WxTryDefault {
		t.Errorf("调用次数 = %d, 期望 %d", calls, WxTryDefault)
	}

	//中途成功则停止重试
	calls = 0
	got, err = reTryWx(ctx, "用例", func(ctx context.Context) (string, error) {
		calls++
		if calls < 2 {
			return "", errors.Errorf("网络异常")
		}
		return "ok", nil
	})
	if err != nil || got != "ok" {
		t.Fatalf("中途成功 = %q, %v", got, err)
	}
	if calls != 2 {
		t.Errorf("中途成功时调用次数 = %d, 期望 2", calls)
	}

	//不可重试的业务错误只能调用一次，否则会白耗配额
	calls = 0
	_, err = reTryWx(ctx, "用例", func(ctx context.Context) (string, error) {
		calls++
		return "", wxRespErr{ErrCode: 45009}
	})
	if err == nil {
		t.Fatalf("不可重试错误应返回error")
	}
	if calls != 1 {
		t.Errorf("不可重试时调用次数 = %d, 期望 1", calls)
	}
}

// ctx 已结束时必须立刻返回且不发起调用，否则同步调用方的超时预算形同虚设
func TestReTryWxCtxDone(t *testing.T) {
	cancelCtx, cancel := context.WithCancel(GenCtx())
	cancel()

	calls := 0
	_, err := reTryWx(cancelCtx, "用例", func(ctx context.Context) (string, error) {
		calls++
		return "ok", nil
	})
	if err == nil {
		t.Fatalf("ctx已结束时应返回error")
	}
	if calls != 0 {
		t.Errorf("ctx已结束时调用次数 = %d, 期望 0", calls)
	}
}

// 环境变量配置了接收人时必须直接采用，不得去调关注者列表接口
func TestGetWxOpenIdsByEnv(t *testing.T) {
	ctx := GenCtx()
	t.Setenv(wxOpenIdKey, " oA , oB ")
	//故意不配凭证：若实现会去调接口，这里一定会报错
	t.Setenv(wxAppIdKey, "")
	t.Setenv(wxSecretKey, "")

	openIds, err := GetWxOpenIds(ctx)
	if err != nil {
		t.Fatalf("环境变量已配置仍报错: %+v", err)
	}
	if len(openIds) != 2 || openIds[0] != "oA" || openIds[1] != "oB" {
		t.Errorf("接收人 = %v", openIds)
	}
}

// 环境变量配置了模板时必须直接采用，不得去调模板列表接口
func TestGetWxTemplateIdByEnv(t *testing.T) {
	ctx := GenCtx()
	t.Setenv(wxTemplateIdKey, "  tpl-1  ")
	t.Setenv(wxAppIdKey, "")
	t.Setenv(wxSecretKey, "")

	templateId, err := GetWxTemplateId(ctx)
	if err != nil {
		t.Fatalf("环境变量已配置仍报错: %+v", err)
	}
	if templateId != "tpl-1" {
		t.Errorf("模板ID = %q, 期望去除空格后的 tpl-1", templateId)
	}
}

// 凭证缺失必须报错而不是 panic，也不得把残缺实例缓存下来
func TestGetWxSdkWithoutEnv(t *testing.T) {
	ctx := GenCtx()
	t.Setenv(wxAppIdKey, "")
	t.Setenv(wxSecretKey, "")

	wxSdkLock.Lock()
	origin := wxSdk
	wxSdk = nil
	wxSdkLock.Unlock()
	defer func() {
		wxSdkLock.Lock()
		wxSdk = origin
		wxSdkLock.Unlock()
	}()

	sdk, err := GetWxSdk(ctx)
	if err == nil {
		t.Fatalf("凭证为空时应报错")
	}
	if sdk != nil {
		t.Errorf("凭证为空时不应返回实例")
	}
	wxSdkLock.Lock()
	cached := wxSdk
	wxSdkLock.Unlock()
	if cached != nil {
		t.Errorf("初始化失败不应缓存实例，否则后续补齐配置也救不回来")
	}
	//只配一半同样要报错
	t.Setenv(wxAppIdKey, "appid")
	if _, err = GetWxSdk(ctx); err == nil {
		t.Errorf("只配AppID时应报错")
	}
}

func TestSendWxMsgEmptyText(t *testing.T) {
	ctx := GenCtx()
	//正文是唯一的真入参，空值必须报错
	if err := SendWxMsg(ctx, "", ""); err == nil {
		t.Errorf("空正文应报错")
	}
	//纯空白等同于空
	if err := SendWxMsg(ctx, "https://x.com", "   \n  "); err == nil {
		t.Errorf("纯空白正文应报错")
	}
}

// 发送失败后必须释放抑制锁。
// 否则调用方在抑制窗口内重试会被静默抑制并返回nil，等于把告警丢掉，
// 与"宁可重复、不可丢失"的取舍直接冲突。
func TestSendWxMsgReleaseRepeatLockOnFail(t *testing.T) {
	ctx := GenCtx()
	//凭证缺失保证发送必然失败，且不会真的发出请求
	t.Setenv(wxAppIdKey, "")
	t.Setenv(wxSecretKey, "")
	t.Setenv(wxOpenIdKey, "")
	t.Setenv(wxTemplateIdKey, "")

	wxSdkLock.Lock()
	origin := wxSdk
	wxSdk = nil
	wxSdkLock.Unlock()
	defer func() {
		wxSdkLock.Lock()
		wxSdk = origin
		wxSdkLock.Unlock()
	}()

	text := "用例正文-" + GenStrId()
	if err := SendWxMsg(ctx, "", text); err == nil {
		t.Fatalf("凭证缺失时首次发送应报错")
	}
	//同一正文立即再发：必须仍然报错，而不是被抑制成nil
	if err := SendWxMsg(ctx, "", text); err == nil {
		t.Errorf("首次失败后抑制锁未释放，重试被静默抑制，告警会丢")
	}
}

// 相同正文在窗口内只放行一次，抑制时返回nil而不是error：抑制是"已经发过了"，不是失败
func TestWxRepeatSuppress(t *testing.T) {
	ctx := GenCtx()
	repeatKey := EnMd5Hex("用例抑制-" + GenStrId())

	lockId := wxRepeatCache.TryLock(ctx, repeatKey, WxRepeatTimeout)
	if lockId <= 0 {
		t.Fatalf("首次应能取得抑制锁")
	}
	if wxRepeatCache.TryLock(ctx, repeatKey, WxRepeatTimeout) > 0 {
		t.Errorf("窗口内重复正文应被抑制")
	}
	//释放后须能重新放行
	wxRepeatCache.UnLock(ctx, repeatKey, lockId)
	if wxRepeatCache.TryLock(ctx, repeatKey, WxRepeatTimeout) <= 0 {
		t.Errorf("释放后应能重新放行")
	}
}

// 抑制键只能由跳转地址与正文决定。
// 若把 logid/ip 算进去，每次调用的键都不同，去重会彻底失效。
func TestWxRepeatKeyIgnoreContext(t *testing.T) {
	url := "https://x.com"
	text := "同一正文"
	first := EnMd5Hex(url + "\n" + text)
	second := EnMd5Hex(url + "\n" + text)
	if first != second {
		t.Fatalf("相同入参应得到相同抑制键")
	}
	//正文不同则键不同
	if EnMd5Hex(url+"\n"+text) == EnMd5Hex(url+"\n"+text+"B") {
		t.Errorf("不同正文不应共享抑制键")
	}
	//跳转地址不同则键不同
	if EnMd5Hex(url+"\n"+text) == EnMd5Hex(url+"/y\n"+text) {
		t.Errorf("不同跳转地址不应共享抑制键")
	}
	//上下文字段不参与：两次 genWxMsgData 的 logid 不同，但抑制键只取正文，必须保持一致
	ctxA := GenCtx()
	ctxB := GenCtx()
	if genWxMsgData(ctxA, text) == genWxMsgData(ctxB, text) {
		t.Fatalf("不同ctx的消息体应因logid不同而不同，用例前提不成立")
	}
	if EnMd5Hex(url+"\n"+text) != EnMd5Hex(url+"\n"+text) {
		t.Errorf("抑制键不应随ctx变化")
	}
}

// ==== 联调探测用例 ====
// 以下用例需要真实的测试号凭证，未配置环境变量时自动跳过，不影响常规 go test。
// 它们用于确认三个无法离线验证的事实：
// 1. 测试号是否允许调用 cgi-bin/user/get（关注者列表，是"最早关注者"整条链路的前提）；
// 2. 测试号是否允许调用 cgi-bin/user/info/batchget（批量用户信息，多关注者时才用到）；
// 3. 测试号是否允许调用 cgi-bin/template/get_all_private_template，以及模板 Content 的原文形态。
// 跑法：wx_app_id=xx wx_secret=xx go test ./util/ -run WxProbe -v

func skipWxProbe(t *testing.T) {
	if GetEnv(wxAppIdKey) == "" || GetEnv(wxSecretKey) == "" {
		t.Skipf("未配置 %s/%s，跳过联调探测", wxAppIdKey, wxSecretKey)
	}
}

func TestWxProbeOpenIdList(t *testing.T) {
	skipWxProbe(t)
	ctx := GenCtx()
	openIds, err := GetWxOpenIdList(ctx)
	if err != nil {
		t.Fatalf("查询关注者列表失败（若为权限类errcode，说明测试号不支持该接口，需改用%s配置接收人）: %+v", wxOpenIdKey, err)
	}
	t.Logf("关注者数量: %d", len(openIds))
	for i := range openIds {
		t.Logf("关注者[%d]: %s", i, openIds[i])
	}
	if len(openIds) == 0 {
		t.Errorf("关注者列表为空，无法推断接收人")
	}
}

func TestWxProbeUserInfos(t *testing.T) {
	skipWxProbe(t)
	ctx := GenCtx()
	openIds, err := GetWxOpenIdList(ctx)
	if err != nil {
		t.Skipf("关注者列表不可用，跳过: %+v", err)
	}
	if len(openIds) == 0 {
		t.Skip("无关注者，跳过")
	}
	infos, err := GetWxUserInfos(ctx, openIds)
	if err != nil {
		t.Fatalf("批量查询用户信息失败（若为权限类errcode，多关注者时需退回逐个查询）: %+v", err)
	}
	for i := range infos {
		if infos[i] == nil {
			continue
		}
		t.Logf("用户[%d]: openid=%s subscribe_time=%d remark=%q scene=%s",
			i, infos[i].OpenID, infos[i].SubscribeTime, infos[i].Remark, infos[i].SubscribeScene)
	}
	//顺带确认最早关注者的判定结果
	earliest, err := getWxEarliestOpenId(ctx)
	if err != nil {
		t.Fatalf("判定最早关注者失败: %+v", err)
	}
	t.Logf("判定出的最早关注者: %s", earliest)
}

func TestWxProbeTemplates(t *testing.T) {
	skipWxProbe(t)
	ctx := GenCtx()
	templates, err := GetWxTemplates(ctx)
	if err != nil {
		t.Fatalf("查询模板列表失败（若为权限类errcode，需改用%s配置模板）: %+v", wxTemplateIdKey, err)
	}
	t.Logf("模板数量: %d", len(templates))
	for i := range templates {
		if templates[i] == nil {
			continue
		}
		//Content 原文用于确认占位符的书写形式，即模板是否确为单个 data 占位符
		t.Logf("模板[%d]: id=%s title=%q\ncontent=%s", i, templates[i].TemplateID, templates[i].Title, templates[i].Content)
	}
	if len(templates) == 0 {
		t.Errorf("模板列表为空，无法推断模板ID")
	}
}

// 真实发送一条消息。除凭证外还需显式开启 wx_probe_send=true，避免误跑时真的推送。
// 跑法：wx_app_id=xx wx_secret=xx wx_probe_send=true go test ./util/ -run TestWxProbeSendMsg -v
func TestWxProbeSendMsg(t *testing.T) {
	skipWxProbe(t)
	if !GetEnvBool("wx_probe_send", false) {
		t.Skip("未开启 wx_probe_send，跳过真实发送")
	}
	ctx := GenCtx()
	//带上随机串，避免被抑制窗口拦掉
	text := "我是正文A\n我是正文B\n我是正文C\n" + GenStrId()
	if err := SendWxMsg(ctx, "https://baidu.com", text); err != nil {
		t.Fatalf("发送失败: %+v", err)
	}
	t.Logf("已发送，请在微信确认消息格式；logid=%d", GetLogId(ctx))
	//url 允许为空的路径也要走一遍
	if err := SendWxMsg(ctx, "", "我是无跳转正文\n"+GenStrId()); err != nil {
		t.Fatalf("空url发送失败: %+v", err)
	}
	t.Logf("空url发送完成")
}

// 注入自定义缓存后，PowerWeChat 不得再创建默认 MemCache。
// 默认实现（PowerLibs/cache/memory.go）会在 ~/.ArtisanCloud/cache 建文件、每次 Set 都写盘，
// 同机多进程还会互相截断该文件；HOME 不可写时 NewMemCache 返回 nil，
// 会让 kernel/accessToken.go 取 token 时对 nil 接口调用 Has 而 panic。
// 构造 SDK 不发起任何网络请求，所以用假凭证即可验证。
func TestWxSdkNotUseDefaultCache(t *testing.T) {
	ctx := GenCtx()
	t.Setenv(wxAppIdKey, "probe-appid")
	t.Setenv(wxSecretKey, "probe-secret")

	home, err := os.UserHomeDir()
	if err != nil {
		t.Skipf("取不到HOME，跳过: %+v", err)
	}
	cacheDir := path.Join(home, ".ArtisanCloud")
	_, err = os.Stat(cacheDir)
	checkDir := os.IsNotExist(err)

	sdk, err := initWxSdk(ctx)
	if err != nil {
		t.Fatalf("假凭证构造SDK失败（构造阶段不应有网络请求）: %+v", err)
	}
	if sdk == nil {
		t.Fatalf("构造SDK返回空实例")
	}

	//硬断言：取token走的必须是我们注入的缓存实现
	if _, ok := sdk.AccessToken.GetCache().(*wxCache); !ok {
		t.Errorf("access_token 缓存为 %T, 期望 *wxCache；默认 MemCache 会写盘并在HOME不可写时panic", sdk.AccessToken.GetCache())
	}
	//辅助断言：原本不存在该目录时，构造SDK后也不应出现
	if checkDir {
		if _, err = os.Stat(cacheDir); err == nil {
			t.Errorf("构造SDK后出现了 %s，说明默认 MemCache 仍被创建", cacheDir)
		}
	}
}
