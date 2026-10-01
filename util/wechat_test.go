package util

import (
	"strings"
	"testing"
	"time"
)

// 这些默认值影响同步调用方被阻塞多久与消息是否被微信拒绝，改动需谨慎
func TestWxConstants(t *testing.T) {
	if wxTimeout != time.Second*10 {
		t.Errorf("wxTimeout = %v", wxTimeout)
	}
	//模板正文里日志行的字段名是 log，与日志字段 LogIdKey(logid) 不同，不能混用
}

// stable_token 模式必须真的生效：
// 走老的 cgi-bin/token 接口时多进程取凭证会互相顶掉，而我们依赖它来避免这件事。
func TestWxSdkStableTokenMode(t *testing.T) {
	ctx := GenCtx()
	t.Setenv(wxAppIdKey, "probe-appid")
	t.Setenv(wxSecretKey, "probe-secret")
	//SDK默认缓存会在HOME下建 .ArtisanCloud 目录并落盘，隔离到临时目录避免污染
	t.Setenv("HOME", t.TempDir())
	defer resetWxSdk(t)()

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
func TestGetWxOpenIdByEnv(t *testing.T) {
	ctx := GenCtx()
	t.Setenv(wxOpenIdKey, " oA , oB ")
	//故意不配凭证：若实现会去调接口，这里一定会报错
	t.Setenv(wxAppIdKey, "")
	t.Setenv(wxSecretKey, "")

	openIds, err := GetWxOpenId(ctx)
	if err != nil {
		t.Fatalf("环境变量已配置仍报错: %+v", err)
	}
	//逗号分隔、去空格
	if len(openIds) != 2 || openIds[0] != "oA" || openIds[1] != "oB" {
		t.Errorf("接收人 = %v", openIds)
	}

	//空项不得产出空字符串元素，否则会向空openid发消息
	t.Setenv(wxOpenIdKey, " oA ,, oB ,")
	openIds, err = GetWxOpenId(ctx)
	if err != nil {
		t.Fatalf("%+v", err)
	}
	if len(openIds) != 2 || openIds[0] != "oA" || openIds[1] != "oB" {
		t.Errorf("含空项时 = %v", openIds)
	}

	//全是空项等同于未配置，会回落到查关注者，此时无凭证必然报错
	t.Setenv(wxOpenIdKey, "  ,  ,")
	defer resetWxSdk(t)()
	if _, err = GetWxOpenId(ctx); err == nil {
		t.Errorf("全空项应回落到查关注者并因无凭证报错")
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

// 缓存只包住查接口那一段：命中时不得穿透到微信接口，env 已配置时不得走缓存，查询失败不得写缓存
func TestGetWxOpenIdCache(t *testing.T) {
	ctx := GenCtx()
	//无凭证，一旦穿透到接口必然报错，以此反证是否命中缓存
	t.Setenv(wxAppIdKey, "")
	t.Setenv(wxSecretKey, "")
	t.Setenv(wxOpenIdKey, "")
	defer resetWxSdk(t)()
	defer cacheWxOpenId.Del(ctx, wxOpenIdKey)

	//查询失败不得写缓存，否则一次网络抖动会让后续一小时的告警全部失败
	if _, err := GetWxOpenId(ctx); err == nil {
		t.Fatalf("无凭证时应报错")
	}
	if _, ok := cacheWxOpenId.Get(ctx, wxOpenIdKey); ok {
		t.Fatalf("查询失败后写入了缓存")
	}
	//第二次仍须报错，而不是命中脏缓存
	if _, err := GetWxOpenId(ctx); err == nil {
		t.Errorf("第二次仍应报错")
	}

	//预置缓存模拟此前已查询成功：此时无凭证也应能拿到结果
	cacheWxOpenId.Set(ctx, wxOpenIdKey, wxCacheTimeout, []string{"oCached"})
	openIds, err := GetWxOpenId(ctx)
	if err != nil {
		t.Fatalf("命中缓存时不应穿透到接口: %+v", err)
	}
	if len(openIds) != 1 || openIds[0] != "oCached" {
		t.Errorf("命中缓存 = %v", openIds)
	}

	//env 已配置时必须用 env 的值，不能被缓存覆盖
	t.Setenv(wxOpenIdKey, "oEnv")
	openIds, err = GetWxOpenId(ctx)
	if err != nil {
		t.Fatalf("%+v", err)
	}
	if len(openIds) != 1 || openIds[0] != "oEnv" {
		t.Errorf("env 应优先于缓存, got %v", openIds)
	}
}

func TestGetWxTemplateIdCache(t *testing.T) {
	ctx := GenCtx()
	t.Setenv(wxAppIdKey, "")
	t.Setenv(wxSecretKey, "")
	t.Setenv(wxTemplateIdKey, "")
	defer resetWxSdk(t)()
	defer cacheWxTemplateId.Del(ctx, wxTemplateIdKey)

	if _, err := GetWxTemplateId(ctx); err == nil {
		t.Fatalf("无凭证时应报错")
	}
	if _, ok := cacheWxTemplateId.Get(ctx, wxTemplateIdKey); ok {
		t.Fatalf("查询失败后写入了缓存")
	}
	if _, err := GetWxTemplateId(ctx); err == nil {
		t.Errorf("第二次仍应报错")
	}

	cacheWxTemplateId.Set(ctx, wxTemplateIdKey, wxCacheTimeout, "tplCached")
	templateId, err := GetWxTemplateId(ctx)
	if err != nil {
		t.Fatalf("命中缓存时不应穿透到接口: %+v", err)
	}
	if templateId != "tplCached" {
		t.Errorf("命中缓存 = %q", templateId)
	}

	t.Setenv(wxTemplateIdKey, "tplEnv")
	templateId, err = GetWxTemplateId(ctx)
	if err != nil {
		t.Fatalf("%+v", err)
	}
	if templateId != "tplEnv" {
		t.Errorf("env 应优先于缓存, got %q", templateId)
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
	earliest, err := GetWxOpenId(ctx)
	if err != nil {
		t.Fatalf("判定最早关注者失败: %+v", err)
	}
	t.Logf("判定出的接收人: %v", earliest)
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
// 只发一条：微信对同一接收人同一模板有 60 秒重复过滤（errcode 40241），一次跑两条第二条必被拒。
// 跳转地址取 wx_probe_url，留空即覆盖「url 允许为空」那条路径，两条路径需间隔 60 秒分两次跑。
// 跑法：wx_app_id=xx wx_secret=xx wx_probe_send=true go test ./util/ -run TestWxProbeSendMsg -v -count=1
func TestWxProbeSendMsg(t *testing.T) {
	skipWxProbe(t)
	if !GetEnvBool("wx_probe_send", false) {
		t.Skip("未开启 wx_probe_send，跳过真实发送")
	}
	ctx := GenCtx()
	url := GetEnv("wx_probe_url")
	text := "我是正文A\n我是正文B\n我是正文C\n" + GenStrId()
	if err := SendWxMsg(ctx, url, text); err != nil {
		t.Fatalf("发送失败（errcode 40241 说明 60 秒内已发过，等一分钟再试）: %+v", err)
	}
	t.Logf("已发送，请在微信确认消息格式；url=%q logid=%d", url, GetLogId(ctx))
}
