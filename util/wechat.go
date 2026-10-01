package util

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/ArtisanCloud/PowerWeChat/v3/src/kernel"
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

// 重试间隔，节奏对齐 httpClient 的 {1s,2s,2s}；条数按"总尝试次数-1"给够
var wxSleeps = []time.Duration{time.Second, time.Second * 2}

// 元数据各自独立缓存实例：Fetch 持写锁，分开后接收人与模板的解析互不阻塞
var wxOpenIdCache = NewLocalCache[[]string]()
var wxTemplateIdCache = NewLocalCache[string]()
var wxRepeatCache = NewLocalCache[int]()

// wxRespErr 微信业务返回码错误。
// PowerWeChat 只把 errcode 用于判断 token 是否失效（kernel/baseClient.go 的 CheckTokenNeedRefresh），
// errcode≠0 时 Go error 仍为 nil，所以必须显式校验并转成 error，否则失败会被静默吞掉。
type wxRespErr struct {
	ErrCode int
	ErrMsg  string
}

func (this wxRespErr) Error() string {
	return fmt.Sprintf("errcode: %d, errmsg: %s", this.ErrCode, this.ErrMsg)
}

// wxCache 替换 PowerWeChat 的默认缓存。
// 默认实现（PowerLibs/cache/memory.go）有四处不可接受的行为：
// 1. 每次 Set 都把整个缓存 SaveFile 写盘；
// 2. 缓存文件固定落在 ~/.ArtisanCloud/cache，同机多进程写同一文件且启动时互相截断；
// 3. HOME 不可写时 NewMemCache 返回 nil，GetCache() 拿到 nil 接口后调用 Has 会 panic；
// 4. Add/AddNX/Remember 均为空实现。
// 注入本实现后 NewInteractsWithCache 不再走 createDefaultCache，上述问题一次性全部规避。
//
// 注意：Get 必须返回 map[string]interface{}。
// PowerWeChat 在 kernel/accessToken.go 中对缓存值做了无保护类型断言 value.(map[string]interface{})，
// 默认实现之所以不 panic，正是因为它 Set 时 json.Marshal、Get 时 json.Unmarshal。
// 本实现必须保持同样的 JSON 往返语义，不能直接存取结构体指针。
type wxCache struct {
	cache LocalCache[[]byte]
}

var _ kernel.CacheInterface = (*wxCache)(nil)

func newWxCache() *wxCache {
	var value wxCache
	value.cache = NewLocalCache[[]byte]()
	return &value
}

func (this *wxCache) Get(key string, defaultValue interface{}) (interface{}, error) {
	data, ok := this.cache.Get(context.Background(), key)
	if !ok || len(data) == 0 {
		return defaultValue, nil
	}
	var value interface{}
	err := JsonData2Struct(data, &value)
	if err != nil {
		return defaultValue, err
	}
	return value, nil
}

func (this *wxCache) Set(key string, value interface{}, expires time.Duration) error {
	data := JsonStruct2Data(value)
	if len(data) == 0 {
		return errors.Errorf("微信SDK缓存，序列化为空")
	}
	//PowerWeChat 写入 access_token 时 TTL 等于微信返回的 expires_in，没有留余量，
	//临界点会用到刚失效的凭证。这里提前失效一点，用一次多余的取token换掉一次必然失败的请求。
	if expires > time.Minute*5 {
		expires -= time.Minute * 5
	}
	//go-cache 把 0 当作"使用默认过期时间"，这里兜底成固定时长，避免被默认值意外缩短
	if expires <= 0 {
		expires = WxMetaTimeout
	}
	this.cache.Set(context.Background(), key, expires, data)
	return nil
}

func (this *wxCache) Has(key string) bool {
	data, ok := this.cache.Get(context.Background(), key)
	return ok && len(data) > 0
}

// AddNX/Add/Remember 在 PowerWeChat 中没有任何调用点（access_token 与 jssdk 只用 Has/Get/Set），
// 实现它们等于写死代码，故与默认实现一致留空。
func (this *wxCache) AddNX(key string, value interface{}, ttl time.Duration) bool {
	return false
}
func (this *wxCache) Add(key string, value interface{}, ttl time.Duration) error {
	return nil
}
func (this *wxCache) Remember(key string, ttl time.Duration, callback func() (interface{}, error)) (interface{}, error) {
	return nil, nil
}

var wxSdk *officialAccount.OfficialAccount
var wxSdkLock = &sync.Mutex{}

// GetWxSdk 懒初始化并复用进程内唯一的公众号SDK实例。
// 不在 Init 中预先初始化：go_common 的多数使用方不配置微信，微信能力只在被调用时才应生效。
// 环境变量在运行期变更不会生效，需重启进程。
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
	//PowerWeChat 构造过程中存在多处无保护类型断言，panic 不应击穿到调用方的主业务
	defer Defer(func(panicValue any, stack string) {
		if panicValue != nil {
			logrus.WithContext(ctx).WithFields(logrus.Fields{"panic": panicValue, "stack": stack}).Error("初始化微信SDK，异常")
			sdk = nil
			err = errors.Errorf("初始化微信SDK，异常: %v", panicValue)
		}
	})

	appId := GetEnv(wxAppIdKey)
	secret := GetEnv(wxSecretKey)
	if appId == "" || secret == "" {
		logrus.WithContext(ctx).WithFields(logrus.Fields{}).Error("初始化微信SDK，凭证为空")
		return nil, errors.Errorf("初始化微信SDK，凭证为空: %s/%s", wxAppIdKey, wxSecretKey)
	}

	serverName := GetServerName()
	var cfg officialAccount.UserConfig
	cfg.AppID = appId
	cfg.Secret = secret
	//stable_token 模式：公众号 access_token 是 AppID 级全局凭证，
	//go_common 被多项目引用后每个进程都会独立取 token，普通 token 接口会互相顶掉。
	//ForceRefresh 置假时走 cgi-bin/stable_token 且不下发 force_refresh，各进程拿到同一个仍有效的凭证。
	cfg.StableTokenMode = true
	cfg.ForceRefresh = false
	cfg.Cache = newWxCache()
	cfg.Http.Timeout = WxTimeoutDefault.Seconds()
	//SDK 自身日志只保留错误级，落到项目既有的日志目录布局下。
	//输出与错误必须用不同文件：PowerWeChat 会为两者各建一个 lumberjack 实例，同文件会互相干扰。
	cfg.Log.Stdout = false
	cfg.Log.Level = "error"
	cfg.Log.File = fmt.Sprintf("log/%s/wechat_info.log", serverName)
	cfg.Log.Error = fmt.Sprintf("log/%s/wechat_error.log", serverName)

	logrus.WithContext(ctx).WithFields(logrus.Fields{"appId": appId}).Info("初始化微信SDK")
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

// isWxErrRetry 判断错误是否值得重试，分层依据见 kernel/baseClient.go：
// 传输层：Go error 非空（网络异常、ctx超时、HTTP非200），重试可能成功；
// 凭证层：errcode 40001/40014/42001 由 PowerWeChat 中间件自动刷token并重放一次，这里不再叠加重试；
// 业务层：errcode≠0 时只有系统繁忙值得重试。参数错误重试必然同样失败，配额超限重试会加速配额耗尽；
// 未知码默认不重试，因为配置类与配额类占多数，后果比少试一次更重。
func isWxErrRetry(err error) bool {
	if err == nil {
		return false
	}
	var respErr wxRespErr
	if errors.As(err, &respErr) {
		return respErr.ErrCode == wxErrCodeBusy
	}
	return true
}

func reTryWx[Resp any](ctx context.Context, name string, funz func(ctx context.Context) (Resp, error)) (resp Resp, err error) {
	for i := 0; i < WxTryDefault; i++ {
		if CtxDone(ctx) {
			logrus.WithContext(ctx).WithFields(logrus.Fields{"try": i}).Error(name + "，上下文已结束")
			return resp, errors.Errorf("%s，上下文已结束", name)
		}
		resp, err = funz(ctx)
		if err == nil {
			return resp, nil
		}
		if !isWxErrRetry(err) {
			logrus.WithContext(ctx).WithFields(logrus.Fields{"err": err}).Error(name + "，不可重试")
			return resp, err
		}
		if i >= WxTryDefault-1 {
			break
		}
		logrus.WithContext(ctx).WithFields(logrus.Fields{"err": err, "try": i}).Warn(name + "，重试")
		sleep := time.Duration(0)
		if len(wxSleeps) > 0 {
			sleep = wxSleeps[min(i, len(wxSleeps)-1)]
		}
		SleepWare(ctx, sleep)
	}
	logrus.WithContext(ctx).WithFields(logrus.Fields{"err": err}).Error(name + "，重试耗尽")
	return resp, err
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
