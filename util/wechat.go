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

	wxTimeout      = time.Second * 10
	wxCacheTimeout = time.Hour
)

var wxSdk *officialAccount.OfficialAccount
var wxSdkLock = &sync.Mutex{}
var cacheWxOpenId = NewLocalCache[[]string]()
var cacheWxTemplateId = NewLocalCache[string]()

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
	cfg.Http.Timeout = wxTimeout.Seconds()
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

func GetWxOpenId(ctx context.Context) ([]string, error) {
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

	return cacheWxOpenId.Fetch(ctx, wxOpenIdKey, wxCacheTimeout, func() ([]string, error) {
		openIds, err := GetWxOpenIdList(ctx)
		if err != nil {
			return nil, err
		}
		if len(openIds) == 0 {
			logrus.WithContext(ctx).WithFields(logrus.Fields{}).Error("查询微信用户，无用户")
			return nil, errors.Errorf("查询微信用户，无用户")
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
			logrus.WithContext(ctx).WithFields(logrus.Fields{}).Error("查询微信用户，用户信息为空")
			return nil, errors.Errorf("查询微信用户，用户信息为空")
		}
		return []string{openId}, nil
	})
}

func GetWxOpenIdList(ctx context.Context) ([]string, error) {
	sdk, err := GetWxSdk(ctx)
	if err != nil {
		return nil, err
	}

	resp, err := sdk.User.List(ctx, "")
	if err != nil || resp == nil {
		logrus.WithContext(ctx).WithFields(logrus.Fields{"err": err}).Error("查询微信用户列表，异常")
		return nil, errors.Errorf("查询微信用户列表，异常: %+v", err)
	}
	if resp.ErrCode != 0 {
		logrus.WithContext(ctx).WithFields(logrus.Fields{"errCode": resp.ErrCode, "errMsg": resp.ErrMsg}).Error("查询微信用户列表，返回码异常")
		return nil, errors.Errorf("查询微信用户列表，返回码异常: errcode=%d, errmsg=%s", resp.ErrCode, resp.ErrMsg)
	}
	return Distinct(ctx, resp.Data.OpenID...), nil
}

func GetWxUserInfos(ctx context.Context, openIds []string) ([]*userResponse.ResponseGetUserInfo, error) {
	sdk, err := GetWxSdk(ctx)
	if err != nil {
		return nil, err
	}

	var req userRequest.RequestBatchGetUserInfo
	for _, openId := range openIds {
		req.UserList = append(req.UserList, &userRequest.UserList{Openid: openId})
	}
	resp, err := sdk.User.BatchGet(ctx, &req)
	if err != nil || resp == nil {
		logrus.WithContext(ctx).WithFields(logrus.Fields{"err": err}).Error("查询微信用户信息，异常")
		return nil, errors.Errorf("查询微信用户信息，异常: %+v", err)
	}
	if resp.ErrCode != 0 {
		logrus.WithContext(ctx).WithFields(logrus.Fields{"errCode": resp.ErrCode, "errMsg": resp.ErrMsg}).Error("查询微信用户信息，返回码异常")
		return nil, errors.Errorf("查询微信用户信息，返回码异常: errcode=%d, errmsg=%s", resp.ErrCode, resp.ErrMsg)
	}
	return resp.ResponseGetUserInfo, nil
}

func GetWxTemplateId(ctx context.Context) (string, error) {
	templateId := strings.TrimSpace(GetEnv(wxTemplateIdKey))
	if templateId != "" {
		return templateId, nil
	}

	return cacheWxTemplateId.Fetch(ctx, wxTemplateIdKey, wxCacheTimeout, func() (string, error) {
		templates, err := GetWxTemplates(ctx)
		if err != nil {
			return "", err
		}
		if len(templates) == 0 {
			logrus.WithContext(ctx).WithFields(logrus.Fields{}).Error("查询微信模板，模板列表为空")
			return "", errors.Errorf("查询微信模板，模板列表为空")
		}
		if len(templates) > 1 {
			logrus.WithContext(ctx).WithFields(logrus.Fields{"count": len(templates)}).Warn("查询微信模板，模板不止一个")
		}
		if templates[0] == nil || templates[0].TemplateID == "" {
			logrus.WithContext(ctx).WithFields(logrus.Fields{}).Error("查询微信模板，模板ID为空")
			return "", errors.Errorf("查询微信模板，模板ID为空")
		}
		return templates[0].TemplateID, nil
	})
}

func GetWxTemplates(ctx context.Context) ([]*templateResponse.Template, error) {
	sdk, err := GetWxSdk(ctx)
	if err != nil {
		return nil, err
	}

	resp, err := sdk.TemplateMessage.GetPrivateTemplates(ctx)
	if err != nil || resp == nil {
		logrus.WithContext(ctx).WithFields(logrus.Fields{"err": err}).Error("查询微信模板列表，异常")
		return nil, errors.Errorf("查询微信模板列表，异常: %+v", err)
	}
	if resp.ErrCode != 0 {
		logrus.WithContext(ctx).WithFields(logrus.Fields{"errCode": resp.ErrCode, "errMsg": resp.ErrMsg}).Error("查询微信模板列表，返回码异常")
		return nil, errors.Errorf("查询微信模板列表，返回码异常: errcode=%d, errmsg=%s", resp.ErrCode, resp.ErrMsg)
	}
	return resp.TemplateList, nil
}

func SendWxTemplateMsg(ctx context.Context, req templateRequest.RequestTemlateMessage) (*templateResponse.ResponseTemplateSend, error) {
	sdk, err := GetWxSdk(ctx)
	if err != nil {
		return nil, err
	}

	resp, err := sdk.TemplateMessage.Send(ctx, &req)
	if err != nil || resp == nil {
		logrus.WithContext(ctx).WithFields(logrus.Fields{"err": err}).Error("发送微信模板消息，异常")
		return nil, errors.Errorf("发送微信模板消息，异常: %+v", err)
	}
	if resp.ErrCode != 0 {
		logrus.WithContext(ctx).WithFields(logrus.Fields{"errCode": resp.ErrCode, "errMsg": resp.ErrMsg}).Error("发送微信模板消息，返回码异常")
		return nil, errors.Errorf("发送微信模板消息，返回码异常: errcode=%d, errmsg=%s", resp.ErrCode, resp.ErrMsg)
	}
	return resp, nil
}

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

	if _, ok := ctx.Deadline(); !ok {
		cancelCtx, cancel := context.WithTimeout(ctx, wxTimeout)
		defer CancelCtx(cancel)
		ctx = cancelCtx
	}

	openIds, err := GetWxOpenId(ctx)
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
	if templateId == "" {
		logrus.WithContext(ctx).WithFields(logrus.Fields{}).Error("发送微信消息，模板为空")
		return errors.Errorf("发送微信消息，模板为空")
	}

	logId := ""
	if value := GetLogId(ctx); value > 0 {
		logId = Int2Str(value)
	}
	hashMap := power.HashMap{}
	hashMap["sn"] = map[string]string{"value": GetServerName()}
	hashMap["ip"] = map[string]string{"value": GetIP()}
	hashMap["lg"] = map[string]string{"value": logId}
	hashMap["te"] = map[string]string{"value": text}

	for _, openId := range openIds {
		var req templateRequest.RequestTemlateMessage
		req.ToUser = openId
		req.TemplateID = templateId
		req.URL = url
		req.Data = &hashMap
		_, eee := SendWxTemplateMsg(ctx, req)
		if eee != nil {
			err = eee
		}
	}
	if err != nil {
		logrus.WithContext(ctx).WithFields(logrus.Fields{"err": err}).Error("发送微信消息，异常")
		return errors.Errorf("发送微信消息，异常: %+v", err)
	}
	logrus.WithContext(ctx).WithFields(logrus.Fields{}).Info("发送微信消息，完成")
	return nil
}
