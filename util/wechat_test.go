package util

import (
	"context"
	"os"
	"path"
	"strings"
	"testing"
	"time"

	"github.com/ArtisanCloud/PowerWeChat/v3/src/kernel/response"
)

// 这些默认值影响同步调用方被阻塞多久与消息是否被微信拒绝，改动需谨慎
func TestWxConstants(t *testing.T) {
	if WxTimeoutDefault != time.Second*10 {
		t.Errorf("WxTimeoutDefault = %v", WxTimeoutDefault)
	}
	if WxTextLenMax <= 0 {
		t.Errorf("WxTextLenMax = %d", WxTextLenMax)
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

// 微信业务失败必须转成 error。
// PowerWeChat 在 errcode≠0 时 Go error 仍为 nil，不显式校验就会出现
// "消息没到但一切正常"的静默失败，这是 msg_gateway 的既有缺陷。
func TestCheckWxErrCode(t *testing.T) {
	ctx := GenCtx()
	if err := checkWxErrCode(ctx, "用例", 0, ""); err != nil {
		t.Errorf("errcode为0时不应报错: %+v", err)
	}
	err := checkWxErrCode(ctx, "用例", 45009, "reach max api daily quota limit")
	if err == nil {
		t.Fatalf("errcode非0时必须报错，否则失败会被静默吞掉")
	}
	//错误信息要带上errcode与errmsg，否则线上无法定位是哪类拒绝
	if !strings.Contains(err.Error(), "45009") {
		t.Errorf("错误应含errcode: %q", err.Error())
	}
	if !strings.Contains(err.Error(), "quota") {
		t.Errorf("错误应含errmsg: %q", err.Error())
	}
	//负数错误码同样要识别（微信用 -1 表示系统繁忙）
	if checkWxErrCode(ctx, "用例", -1, "system error") == nil {
		t.Errorf("负数errcode也必须报错")
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
	if lines[2] != wxLogKey+": "+Int2Str(GetLogId(ctx)) {
		t.Errorf("日志行 = %q", lines[2])
	}
}

// 上下文取不到值时必须留空，不能报错也不能填成 0。
// GetIP 由后台协程异步填充，进程刚启动时必然为空，而那恰是最需要告警的时段。
func TestGenWxMsgDataEmptyContext(t *testing.T) {
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
	if lines[3] != "正文" {
		t.Errorf("正文 = %q", lines[3])
	}
}

func TestCutWxText(t *testing.T) {
	ctx := GenCtx()
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
		t.Errorf("截断后应有标记")
	}
	//边界：恰好等于上限不截断
	exact := strings.Repeat("中", WxTextLenMax)
	if cutWxText(ctx, exact) != exact {
		t.Errorf("恰好等于上限时不应截断")
	}
}

// wxCache 的核心契约：Get 必须返回 map[string]interface{}。
// PowerWeChat 在 kernel/accessToken.go:113 对缓存值做 value.(map[string]interface{}) 无保护断言，
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
	if value, _ = c.Get("absent", "def"); value != "def" {
		t.Errorf("未命中应返回传入的 defaultValue, got %v", value)
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

// stable_token 模式必须真的生效：
// 走老的 cgi-bin/token 接口时多进程取凭证会互相顶掉，而我们依赖它来避免这件事。
func TestWxSdkStableTokenMode(t *testing.T) {
	ctx := GenCtx()
	t.Setenv(wxAppIdKey, "probe-appid")
	t.Setenv(wxSecretKey, "probe-secret")

	sdk, err := initWxSdk(ctx)
	if err != nil {
		t.Fatalf("假凭证构造SDK失败: %+v", err)
	}
	if !sdk.AccessToken.StableTokenMode {
		t.Errorf("StableTokenMode 未生效，多进程取凭证会互相顶掉")
	}
	//ForceRefresh 必须为假，否则每次都强制换新凭证，等于白用了稳定版接口
	if sdk.AccessToken.ForceRefresh {
		t.Errorf("ForceRefresh 应为false，否则仍会顶掉其他进程的凭证")
	}
	if !strings.Contains(sdk.AccessToken.EndpointToGetToken, "stable_token") {
		t.Errorf("取凭证端点 = %q, 期望含 stable_token", sdk.AccessToken.EndpointToGetToken)
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
	defer resetWxSdk(t)()

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

// 不做去重：相同正文连续发送两次必须都走完整流程并各自返回结果，
// 第二次不得被静默抑制成 nil，否则等于把告警丢掉。
func TestSendWxMsgNoRepeatSuppress(t *testing.T) {
	ctx := GenCtx()
	//凭证缺失保证发送必然失败，且不会真的发出请求
	t.Setenv(wxAppIdKey, "")
	t.Setenv(wxSecretKey, "")
	t.Setenv(wxOpenIdKey, "")
	t.Setenv(wxTemplateIdKey, "")
	defer resetWxSdk(t)()

	text := "用例正文-" + GenStrId()
	if err := SendWxMsg(ctx, "", text); err == nil {
		t.Fatalf("凭证缺失时首次发送应报错")
	}
	//同一正文立即再发：必须仍然报错，而不是被抑制成nil
	if err := SendWxMsg(ctx, "", text); err == nil {
		t.Errorf("相同正文第二次被静默抑制了，告警会丢")
	}
}

// resetWxSdk 清空进程内SDK单例，返回用于恢复的函数。
// 单例会跨用例复用，不清空会让后续用例读到前面用例构造的实例。
func resetWxSdk(t *testing.T) func() {
	wxSdkLock.Lock()
	origin := wxSdk
	wxSdk = nil
	wxSdkLock.Unlock()
	return func() {
		wxSdkLock.Lock()
		wxSdk = origin
		wxSdkLock.Unlock()
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
