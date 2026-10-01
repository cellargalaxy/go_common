package util

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/ArtisanCloud/PowerWeChat/v3/src/kernel/power"
	"github.com/ArtisanCloud/PowerWeChat/v3/src/officialAccount"
	templateRequest "github.com/ArtisanCloud/PowerWeChat/v3/src/officialAccount/templateMessage/request"
	templateResponse "github.com/ArtisanCloud/PowerWeChat/v3/src/officialAccount/templateMessage/response"
	userRequest "github.com/ArtisanCloud/PowerWeChat/v3/src/officialAccount/user/request"
	userResponse "github.com/ArtisanCloud/PowerWeChat/v3/src/officialAccount/user/response"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

const (
	wxAppIdKey      = "wx_app_id"
	wxSecretKey     = "wx_secret"
	wxOpenIdKey     = "wx_open_id"
	wxTemplateIdKey = "wx_template_id"
)

const (
	wxDataKey = "data"
	wxLogKey  = "log"
)

const (
	WxTimeoutDefault = time.Second * 10
)

var wxSdk *officialAccount.OfficialAccount
var wxSdkLock = &sync.Mutex{}

func GetWxSdk(ctx context.Context) (*officialAccount.OfficialAccount, error) {
	wxSdkLock.Lock()
	defer wxSdkLock.Unlock()

	if wxSdk != nil {
		return wxSdk, nil
	}
	sdk, err := initWxSdk(ctx)
	if err != nil {
		return nil, err
	}
	wxSdk = sdk
	return wxSdk, nil
}

func initWxSdk(ctx context.Context) (sdk *officialAccount.OfficialAccount, err error) {
	defer Defer(func(panic any, stack string) {
		if panic != nil {
			logrus.WithContext(ctx).WithFields(logrus.Fields{"panic": panic, "stack": stack}).Error("初始化微信SDK，异常")
			sdk = nil
			err = errors.Errorf("初始化微信SDK，异常: %v", panic)
		}
	})

	appId := GetEnv(wxAppIdKey)
	if appId == "" {
		logrus.WithContext(ctx).WithFields(logrus.Fields{}).Error("初始化微信SDK，appId为空")
		return nil, errors.Errorf("初始化微信SDK，appId为空")
	}
	secret := GetEnv(wxSecretKey)
	if secret == "" {
		logrus.WithContext(ctx).WithFields(logrus.Fields{}).Error("初始化微信SDK，secret为空")
		return nil, errors.Errorf("初始化微信SDK，secret为空")
	}

	sn := GetServerName()
	var cfg officialAccount.UserConfig
	cfg.AppID = appId
	cfg.Secret = secret
	//走 stable_token 取凭证且只问不换新，避免多进程共用同一公众号时互相顶掉凭证
	cfg.StableTokenMode = true
	cfg.ForceRefresh = false
	cfg.Cache = newWxCache()
	cfg.Http.Timeout = WxTimeoutDefault.Seconds()
	//输出与错误须用不同文件，SDK会为两者各建一个lumberjack实例
	cfg.Log.Stdout = false
	cfg.Log.Level = "error"
	cfg.Log.File = fmt.Sprintf("log/%s/wechat_info.log", sn)
	cfg.Log.Error = fmt.Sprintf("log/%s/wechat_error.log", sn)

	logrus.WithContext(ctx).WithFields(logrus.Fields{}).Info("初始化微信SDK")
	sdk, err = officialAccount.NewOfficialAccount(&cfg)
	if err != nil {
		logrus.WithContext(ctx).WithFields(logrus.Fields{"err": err}).Error("初始化微信SDK，异常")
		return nil, errors.Errorf("初始化微信SDK，异常: %+v", err)
	}
	if sdk == nil {
		logrus.WithContext(ctx).WithFields(logrus.Fields{}).Error("初始化微信SDK，实例为空")
		return nil, errors.Errorf("初始化微信SDK，实例为空")
	}
	return sdk, nil
}

func GetWxOpenIds(ctx context.Context) ([]string, error) {
	text := GetEnv(wxOpenIdKey)
	items := strings.Split(text, ",")
	openIds := make([]string, 0, len(items))
	for i := range items {
		item := strings.TrimSpace(items[i])
		if item == "" {
			continue
		}
		openIds = append(openIds, item)
	}
	if len(openIds) > 0 {
		return openIds, nil
	}

	openIds, err := GetWxOpenIdList(ctx)
	if err != nil {
		return nil, err
	}
	if len(openIds) == 0 {
		logrus.WithContext(ctx).WithFields(logrus.Fields{}).Error("查询关注者，无关注者")
		return nil, errors.Errorf("查询关注者，无关注者")
	}
	if len(openIds) == 1 {
		return openIds, nil
	}
	infos, err := GetWxUserInfos(ctx, openIds)
	if err != nil {
		return nil, err
	}
	openId := ""
	subscribeTime := 0
	for i := range infos {
		info := infos[i]
		if info == nil || info.OpenID == "" {
			continue
		}
		if subscribeTime == 0 || info.SubscribeTime < subscribeTime {
			openId = info.OpenID
			subscribeTime = info.SubscribeTime
		}
	}
	if openId == "" {
		logrus.WithContext(ctx).WithFields(logrus.Fields{}).Error("查询关注者，用户信息为空")
		return nil, errors.Errorf("查询关注者，用户信息为空")
	}
	return []string{openId}, nil
}

func GetWxOpenIdList(ctx context.Context) ([]string, error) {
	sdk, err := GetWxSdk(ctx)
	if err != nil {
		return nil, err
	}

	var openIds []string
	nextOpenId := ""

	resp, err := sdk.User.List(ctx, nextOpenId)
	if err != nil || resp == nil {
		logrus.WithContext(ctx).WithFields(logrus.Fields{"err": err}).Error(name + "，异常")
		return nil, errors.Errorf("%s，异常: %+v", name, err)
	}
	if err = checkWxErrCode(ctx, name, resp.ErrCode, resp.ErrMsg); err != nil {
		return nil, err
	}
	openIds = append(openIds, resp.Data.OpenID...)
	//游标为空或未推进都说明已取完，后者同时防御死循环
	if resp.NextOpenID == "" || resp.NextOpenID == nextOpenId || len(resp.Data.OpenID) == 0 {
		break
	}
	nextOpenId = resp.NextOpenID
	return Distinct(ctx, openIds...), nil
}

func GetWxUserInfos(ctx context.Context, openIds []string) ([]*userResponse.ResponseGetUserInfo, error) {
	sdk, err := GetWxSdk(ctx)
	if err != nil {
		return nil, err
	}

	var infos []*userResponse.ResponseGetUserInfo
	var req userRequest.RequestBatchGetUserInfo
	for _, openId := range openIds[i:min(i+wxUserBatchMax, len(openIds))] {
		req.UserList = append(req.UserList, &userRequest.UserList{Openid: openId})
	}
	resp, err := sdk.User.BatchGet(ctx, &req)
	if err != nil || resp == nil {
		logrus.WithContext(ctx).WithFields(logrus.Fields{"err": err}).Error(name + "，异常")
		return nil, errors.Errorf("%s，异常: %+v", name, err)
	}
	if err = checkWxErrCode(ctx, name, resp.ErrCode, resp.ErrMsg); err != nil {
		return nil, err
	}
	infos = append(infos, resp.ResponseGetUserInfo...)
	return infos, nil
}

func GetWxTemplateId(ctx context.Context) (string, error) {
	templateId := strings.TrimSpace(GetEnv(wxTemplateIdKey))
	if templateId != "" {
		return templateId, nil
	}

	templates, err := GetWxTemplates(ctx)
	if err != nil {
		return "", err
	}
	if len(templates) == 0 {
		logrus.WithContext(ctx).WithFields(logrus.Fields{}).Error("查询微信模板，模板列表为空")
		return "", errors.Errorf("查询微信模板，模板列表为空")
	}
	//取第一个而非随机，随机会产生时而成功时而被微信拒绝的非确定性故障
	if len(templates) > 1 {
		logrus.WithContext(ctx).WithFields(logrus.Fields{"count": len(templates)}).Warn("查询微信模板，模板不止一个，将取第一个；如需指定模板请配置" + wxTemplateIdKey)
	}
	if templates[0] == nil || templates[0].TemplateID == "" {
		logrus.WithContext(ctx).WithFields(logrus.Fields{}).Error("查询微信模板，模板ID为空")
		return "", errors.Errorf("查询微信模板，模板ID为空")
	}
	return templates[0].TemplateID, nil
}

func GetWxTemplates(ctx context.Context) ([]*templateResponse.Template, error) {
	name := "查询微信模板列表"
	sdk, err := GetWxSdk(ctx)
	if err != nil {
		return nil, err
	}

	resp, err := sdk.TemplateMessage.GetPrivateTemplates(ctx)
	if err != nil || resp == nil {
		logrus.WithContext(ctx).WithFields(logrus.Fields{"err": err}).Error(name + "，异常")
		return nil, errors.Errorf("%s，异常: %+v", name, err)
	}
	if err = checkWxErrCode(ctx, name, resp.ErrCode, resp.ErrMsg); err != nil {
		return nil, err
	}
	return resp.TemplateList, nil
}

func genWxMsgData(ctx context.Context, text string) string {
	//上下文取不到就留空：GetIP由后台协程异步填充，进程刚启动时必然为空，
	//而启动期恰是最需要告警的时段，不能因上下文缺失阻断发送
	logId := ""
	if value := GetLogId(ctx); value > 0 {
		logId = Int2Str(value)
	}
	var builder strings.Builder
	builder.WriteString(fmt.Sprintf("%s: %s\n", ServerNameKey, GetServerName()))
	builder.WriteString(fmt.Sprintf("%s: %s\n", IpKey, GetIP()))
	builder.WriteString(fmt.Sprintf("%s: %s\n", wxLogKey, logId))
	builder.WriteString(text)
	return builder.String()
}

// SendWxMsg 同步发送一条模板消息。url 允许为空，text 不允许为空。
// sn/ip/log 由函数自动拼接。不重试不去重，失败即返回error，要不要重发由调用方决定。
func SendWxMsg(ctx context.Context, url, text string) (err error) {
	defer Defer(func(panic any, stack string) {
		if panic != nil {
			logrus.WithContext(ctx).WithFields(logrus.Fields{"panic": panic, "stack": stack}).Error("发送微信消息，异常")
			err = errors.Errorf("发送微信消息，异常: %v", panic)
		}
	})

	text = strings.TrimSpace(text)
	if text == "" {
		logrus.WithContext(ctx).WithFields(logrus.Fields{}).Error("发送微信消息，正文为空")
		return errors.Errorf("发送微信消息，正文为空")
	}

	//SDK取token的请求没有客户端级超时，ctx的deadline是同步调用方唯一的兜底，
	//而GenCtx基于Background不带deadline，所以这里补上总预算
	if _, ok := ctx.Deadline(); !ok {
		cancelCtx, cancel := context.WithTimeout(ctx, WxTimeoutDefault)
		defer CancelCtx(cancel)
		ctx = cancelCtx
	}

	openIds, err := GetWxOpenIds(ctx)
	if err != nil {
		return err
	}
	if len(openIds) == 0 {
		logrus.WithContext(ctx).WithFields(logrus.Fields{}).Error("发送微信消息，接收人为空")
		return errors.Errorf("发送微信消息，接收人为空")
	}
	templateId, err := GetWxTemplateId(ctx)
	if err != nil {
		return err
	}

	hashMap := power.HashMap{wxDataKey: map[string]string{"value": genWxMsgData(ctx, cutWxText(ctx, text))}}
	//全部成功才算成功，但都尝试完再聚合错误，不因某个接收人失败而漏发其他人
	errMsgs := make([]string, 0, len(openIds))
	for _, openId := range openIds {
		var req templateRequest.RequestTemlateMessage
		req.ToUser = openId
		req.TemplateID = templateId
		req.URL = url
		req.Data = &hashMap
		_, err = SendWxTemplateMsg(ctx, req)
		if err != nil {
			errMsgs = append(errMsgs, fmt.Sprintf("%s: %v", openId, err))
		}
	}
	if len(errMsgs) > 0 {
		logrus.WithContext(ctx).WithFields(logrus.Fields{"total": len(openIds), "fail": len(errMsgs)}).Error("发送微信消息，异常")
		return errors.Errorf("发送微信消息，异常: %s", strings.Join(errMsgs, "; "))
	}
	logrus.WithContext(ctx).WithFields(logrus.Fields{"total": len(openIds)}).Info("发送微信消息，完成")
	return nil
}

func SendWxTemplateMsg(ctx context.Context, req templateRequest.RequestTemlateMessage) (*templateResponse.ResponseTemplateSend, error) {
	name := "发送微信模板消息"
	sdk, err := GetWxSdk(ctx)
	if err != nil {
		return nil, err
	}

	resp, err := sdk.TemplateMessage.Send(ctx, &req)
	if err != nil || resp == nil {
		logrus.WithContext(ctx).WithFields(logrus.Fields{"err": err}).Error(name + "，异常")
		return nil, errors.Errorf("%s，异常: %+v", name, err)
	}
	if err = checkWxErrCode(ctx, name, resp.ErrCode, resp.ErrMsg); err != nil {
		return nil, err
	}
	return resp, nil
}
