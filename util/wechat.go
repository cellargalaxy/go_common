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
	wxAppIdKey      = "wx_app_id"      //环境变量：公众号AppID
	wxSecretKey     = "wx_secret"      //环境变量：公众号AppSecret
	wxOpenIdKey     = "wx_open_id"     //环境变量：接收人openid，逗号分隔多个；未配置时回退为最早关注者
	wxTemplateIdKey = "wx_template_id" //环境变量：模板ID；未配置时回退为模板列表第一个
)

const (
	wxDataKey = "data" //模板唯一占位符 {{data.DATA}} 的字段名
	wxLogKey  = "log"  //正文中日志ID行的字段名，与 LogIdKey("logid") 不同，按模板约定为 log
)

const (
	WxTimeoutDefault = time.Second * 10 //单条消息的总耗时预算，含重试
	WxTryDefault     = 3                //单次微信接口调用的总尝试次数
	WxMetaTimeout    = time.Hour        //接收人/模板等元数据的缓存时长
	WxRepeatTimeout  = time.Minute * 5  //相同正文的抑制窗口，防错误风暴打爆测试号配额
	WxTextLenMax     = 1000             //正文最大字符数。微信对模板占位符取值的长度上限未确认，此处取保守值
)

const (
	wxErrCodeBusy  = -1  //微信系统繁忙，可重试
	wxUserBatchMax = 100 //批量拉取用户信息的单批上限。微信官方上限未确认，此处取保守值
	wxUserPageMax  = 100 //拉取关注者列表的最大翻页轮次，仅用于防御接口异常导致的死循环
)

var wxOpenIdCache = NewLocalCache[[]string]()
var wxTemplateIdCache = NewLocalCache[string]()
var wxRepeatCache = NewLocalCache[int]()

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
	cfg.StableTokenMode = true
	cfg.ForceRefresh = false
	cfg.Cache = newWxCache()
	cfg.Http.Timeout = WxTimeoutDefault.Seconds()
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

// GetWxOpenIds 解析接收人。环境变量优先，未配置时回退为最早关注者。
func GetWxOpenIds(ctx context.Context) ([]string, error) {
	openIds := splitWxOpenIds(GetEnv(wxOpenIdKey))
	if len(openIds) > 0 {
		return openIds, nil
	}
	//Fetch 在 get 返回错误时不写缓存，所以"空结果判为失败"即可避免空值被缓存住
	return wxOpenIdCache.Fetch(ctx, wxOpenIdKey, WxMetaTimeout, func() ([]string, error) {
		openId, err := getWxEarliestOpenId(ctx)
		if err != nil {
			return nil, err
		}
		return []string{openId}, nil
	})
}

func splitWxOpenIds(text string) []string {
	items := strings.Split(text, ",")
	openIds := make([]string, 0, len(items))
	for i := range items {
		item := strings.TrimSpace(items[i])
		if item == "" {
			continue
		}
		openIds = append(openIds, item)
	}
	return openIds
}

// getWxEarliestOpenId 取最早关注者。
// 测试号只有本人关注，最早关注者即为本人（已与需求方确认）。
// 不依赖关注者列表接口的返回顺序：该顺序是否等于关注顺序没有任何依据，
// 所以多关注者时改用 subscribe_time 显式排序。
func getWxEarliestOpenId(ctx context.Context) (string, error) {
	openIds, err := GetWxOpenIdList(ctx)
	if err != nil {
		return "", err
	}
	if len(openIds) == 0 {
		logrus.WithContext(ctx).WithFields(logrus.Fields{}).Error("查询微信最早关注者，无关注者")
		return "", errors.Errorf("查询微信最早关注者，无关注者")
	}
	if len(openIds) == 1 {
		return openIds[0], nil
	}

	logrus.WithContext(ctx).WithFields(logrus.Fields{"count": len(openIds)}).Warn("查询微信最早关注者，关注者不止一人，将取最早关注者；如需指定接收人请配置" + wxOpenIdKey)
	infos, err := GetWxUserInfos(ctx, openIds)
	if err != nil {
		return "", err
	}

	earliestOpenId := ""
	earliestTime := 0
	for i := range infos {
		info := infos[i]
		if info == nil || info.OpenID == "" {
			continue
		}
		//关注时间更早则取之；同秒关注时取openid字典序最小者，保证同一环境下结果确定、日志可复现
		if earliestOpenId == "" || info.SubscribeTime < earliestTime ||
			(info.SubscribeTime == earliestTime && info.OpenID < earliestOpenId) {
			earliestOpenId = info.OpenID
			earliestTime = info.SubscribeTime
		}
	}
	if earliestOpenId == "" {
		logrus.WithContext(ctx).WithFields(logrus.Fields{}).Error("查询微信最早关注者，用户信息为空")
		return "", errors.Errorf("查询微信最早关注者，用户信息为空")
	}
	return earliestOpenId, nil
}

// GetWxOpenIdList 分页拉取全部关注者openid
func GetWxOpenIdList(ctx context.Context) ([]string, error) {
	//SDK 初始化失败属于配置错误，重试必然同样失败，所以放在重试之外
	sdk, err := GetWxSdk(ctx)
	if err != nil {
		return nil, err
	}
	var openIds []string
	nextOpenId := ""
	for i := 0; i < wxUserPageMax; i++ {
		lastOpenId := nextOpenId
		resp, err := reTryWx(ctx, "查询微信关注者列表", func(ctx context.Context) (*userResponse.ResponseGetUserList, error) {
			resp, err := sdk.User.List(ctx, lastOpenId)
			if err != nil {
				return nil, err
			}
			if resp == nil {
				return nil, errors.Errorf("查询微信关注者列表，响应为空")
			}
			if resp.ErrCode != 0 {
				return nil, wxRespErr{ErrCode: resp.ErrCode, ErrMsg: resp.ErrMsg}
			}
			return resp, nil
		})
		if err != nil {
			return nil, err
		}
		openIds = append(openIds, resp.Data.OpenID...)
		//游标为空或未推进都说明已取完，后者同时防御接口异常导致的死循环
		if resp.NextOpenID == "" || resp.NextOpenID == lastOpenId || len(resp.Data.OpenID) == 0 {
			break
		}
		nextOpenId = resp.NextOpenID
	}
	return Distinct(ctx, openIds...), nil
}

// GetWxUserInfos 分批拉取用户基本信息
func GetWxUserInfos(ctx context.Context, openIds []string) ([]*userResponse.ResponseGetUserInfo, error) {
	//SDK 初始化失败属于配置错误，重试必然同样失败，所以放在重试之外
	sdk, err := GetWxSdk(ctx)
	if err != nil {
		return nil, err
	}
	var infos []*userResponse.ResponseGetUserInfo
	for i := 0; i < len(openIds); i += wxUserBatchMax {
		var req userRequest.RequestBatchGetUserInfo
		for _, openId := range openIds[i:min(i+wxUserBatchMax, len(openIds))] {
			req.UserList = append(req.UserList, &userRequest.UserList{Openid: openId})
		}
		resp, err := reTryWx(ctx, "查询微信用户信息", func(ctx context.Context) (*userResponse.ResponseBatchGetUserInfo, error) {
			resp, err := sdk.User.BatchGet(ctx, &req)
			if err != nil {
				return nil, err
			}
			if resp == nil {
				return nil, errors.Errorf("查询微信用户信息，响应为空")
			}
			if resp.ErrCode != 0 {
				return nil, wxRespErr{ErrCode: resp.ErrCode, ErrMsg: resp.ErrMsg}
			}
			return resp, nil
		})
		if err != nil {
			return nil, err
		}
		infos = append(infos, resp.ResponseGetUserInfo...)
	}
	return infos, nil
}

// GetWxTemplateId 解析模板ID。环境变量优先，未配置时取模板列表第一个。
// 只会配置一个通用模板（已与需求方确认），列表不止一个时取第一个并告警：
// 取第一个在同一环境下行为稳定、日志可复现，随机取则会产生时而成功时而被微信拒绝的非确定性故障。
func GetWxTemplateId(ctx context.Context) (string, error) {
	templateId := strings.TrimSpace(GetEnv(wxTemplateIdKey))
	if templateId != "" {
		return templateId, nil
	}
	return wxTemplateIdCache.Fetch(ctx, wxTemplateIdKey, WxMetaTimeout, func() (string, error) {
		templates, err := GetWxTemplates(ctx)
		if err != nil {
			return "", err
		}
		if len(templates) == 0 {
			logrus.WithContext(ctx).WithFields(logrus.Fields{}).Error("查询微信模板，模板列表为空")
			return "", errors.Errorf("查询微信模板，模板列表为空")
		}
		if len(templates) > 1 {
			logrus.WithContext(ctx).WithFields(logrus.Fields{"count": len(templates)}).Warn("查询微信模板，模板不止一个，将取第一个；如需指定模板请配置" + wxTemplateIdKey)
		}
		if templates[0] == nil || templates[0].TemplateID == "" {
			logrus.WithContext(ctx).WithFields(logrus.Fields{}).Error("查询微信模板，模板ID为空")
			return "", errors.Errorf("查询微信模板，模板ID为空")
		}
		return templates[0].TemplateID, nil
	})
}

// GetWxTemplates 查询账号下的模板列表
func GetWxTemplates(ctx context.Context) ([]*templateResponse.Template, error) {
	//SDK 初始化失败属于配置错误，重试必然同样失败，所以放在重试之外
	sdk, err := GetWxSdk(ctx)
	if err != nil {
		return nil, err
	}
	resp, err := reTryWx(ctx, "查询微信模板列表", func(ctx context.Context) (*templateResponse.ResponseTemplateGetPrivate, error) {
		resp, err := sdk.TemplateMessage.GetPrivateTemplates(ctx)
		if err != nil {
			return nil, err
		}
		if resp == nil {
			return nil, errors.Errorf("查询微信模板列表，响应为空")
		}
		if resp.ErrCode != 0 {
			return nil, wxRespErr{ErrCode: resp.ErrCode, ErrMsg: resp.ErrMsg}
		}
		return resp, nil
	})
	if err != nil {
		return nil, err
	}
	return resp.TemplateList, nil
}

// genWxMsgData 拼接模板唯一占位符 data 的取值，结构为 sn/ip/log 三行上下文加正文。
// 上下文取不到时留空而不报错：GetIP 由 initHttp 启动的后台协程异步填充，进程刚启动时必然为空，
// 未调用 Init 时则恒为空，而启动期恰是最需要告警的时段，不能因上下文缺失阻断发送。
func genWxMsgData(ctx context.Context, text string) string {
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

// cutWxText 截断超长正文。微信对模板占位符取值的长度上限未确认，
// 这里按保守值截断，避免整条消息被微信拒绝导致彻底收不到。
func cutWxText(ctx context.Context, text string) string {
	runes := []rune(text)
	if len(runes) <= WxTextLenMax {
		return text
	}
	logrus.WithContext(ctx).WithFields(logrus.Fields{"len": len(runes), "max": WxTextLenMax}).Warn("发送微信消息，正文超长已截断")
	return string(runes[:WxTextLenMax]) + "...(已截断)"
}

// SendWxMsg 通过公众号模板消息发送一条通知，同步执行，异步由调用方自行决定。
// url 为点击消息后的跳转地址，允许为空；text 为正文，不允许为空。
// sn/ip/log 三行上下文由本函数自动拼接，调用方只需关心正文。
func SendWxMsg(ctx context.Context, url, text string) (err error) {
	var repeatKey string
	var lockId int64
	//微信链路是旁路能力，其内部 panic 不应击穿到调用方的主业务
	defer Defer(func(panicValue any, stack string) {
		if panicValue == nil {
			return
		}
		logrus.WithContext(ctx).WithFields(logrus.Fields{"panic": panicValue, "stack": stack}).Error("发送微信消息，异常")
		err = errors.Errorf("发送微信消息，异常: %v", panicValue)
		if lockId > 0 {
			wxRepeatCache.UnLock(ctx, repeatKey, lockId)
		}
	})

	text = strings.TrimSpace(text)
	if text == "" {
		logrus.WithContext(ctx).WithFields(logrus.Fields{}).Error("发送微信消息，正文为空")
		return errors.Errorf("发送微信消息，正文为空")
	}

	//相同正文在抑制窗口内只发一次，防止错误风暴打爆测试号配额。
	//抑制键只取跳转地址与正文，不含 logid/ip，否则每次都不同，去重会失效。
	repeatKey = EnMd5Hex(fmt.Sprintf("%s\n%s", url, text))
	lockId = wxRepeatCache.TryLock(ctx, repeatKey, WxRepeatTimeout)
	if lockId <= 0 {
		logrus.WithContext(ctx).WithFields(logrus.Fields{}).Info("发送微信消息，重复正文已抑制")
		return nil
	}

	err = sendWxMsg(ctx, url, text)
	if err != nil {
		//失败必须释放抑制锁，否则调用方在窗口内重试会被静默抑制，等于丢告警
		wxRepeatCache.UnLock(ctx, repeatKey, lockId)
	}
	return err
}

func sendWxMsg(ctx context.Context, url, text string) error {
	//取token的请求在 PowerWeChat 内部没有客户端级超时（kernel/accessToken.go 构造 helper.Config 时未设 Timeout），
	//ctx 的 deadline 是同步调用方唯一的兜底，而 GenCtx 基于 Background 并不带 deadline，所以这里必须补上总预算
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
	//全部成功才算成功。逐个接收人都尝试完再聚合错误，不因某个接收人失败而漏发其他人
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

// SendWxTemplateMsg 发送一条模板消息
func SendWxTemplateMsg(ctx context.Context, req templateRequest.RequestTemlateMessage) (*templateResponse.ResponseTemplateSend, error) {
	//SDK 初始化失败属于配置错误，重试必然同样失败，所以放在重试之外
	sdk, err := GetWxSdk(ctx)
	if err != nil {
		return nil, err
	}
	return reTryWx(ctx, "发送微信模板消息", func(ctx context.Context) (*templateResponse.ResponseTemplateSend, error) {
		resp, err := sdk.TemplateMessage.Send(ctx, &req)
		if err != nil {
			return nil, err
		}
		if resp == nil {
			return nil, errors.Errorf("发送微信模板消息，响应为空")
		}
		if resp.ErrCode != 0 {
			return nil, wxRespErr{ErrCode: resp.ErrCode, ErrMsg: resp.ErrMsg}
		}
		return resp, nil
	})
}
