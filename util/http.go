package util

import (
	"context"
	"crypto/tls"
	"fmt"
	"github.com/cellargalaxy/go_common/model"
	"github.com/go-resty/resty/v2"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"net/http"
	"strings"
	"sync"
	"time"
)

const (
	TimeoutDefault   = time.Second * 3
	SleepDefault     = time.Second * 3
	TryDefault       = 3
	UserAgentKey     = "User-Agent"
	UserAgentDefault = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/83.0.4103.116 Safari/537.36"
)

var SpiderSleepsDefault = []time.Duration{0, time.Second * 2, time.Second * 2}
var httpClient *resty.Client
var httpClientOnce sync.Once
var httpClientSpider *resty.Client
var httpClientSpiderOnce sync.Once
var ip string

func initHttp(ctx context.Context) {
	var err error
	_, err = NewDaemonSingleGoPool(ctx, "HttpGetIp", time.Hour, flushHttpGetIp)
	if err != nil {
		panic(err)
	}
}

type HttpResponseInter interface {
	HttpSuccess(ctx context.Context) error
}

func HttpApiWithTry(ctx context.Context, name string, try int, sleeps []time.Duration, response HttpResponseInter, newResponse func() (*resty.Response, error)) error {
	if try < len(sleeps)+1 {
		try = len(sleeps) + 1
	}
	var err error
	for i := 0; i < try; i++ {
		err = HttpApi(ctx, name, response, newResponse)
		if err == nil {
			return nil
		}
		wareSleep := WareDuration(GetSleepTime(sleeps, i))
		logrus.WithContext(ctx).WithFields(logrus.Fields{"err": err, "wareSleep": wareSleep}).Error(genHttpText(ctx, name, nil, "异常", "重试请求"))
		Sleep(ctx, wareSleep)
	}
	return err
}
func HttpApi(ctx context.Context, name string, response HttpResponseInter, newResponse func() (*resty.Response, error)) error {
	resp, err := newResponse()
	body, err := DealHttpResponse(ctx, name, resp, err)
	if err != nil {
		return err
	}
	err = JsonString2Struct(body, response)
	if err != nil {
		logrus.WithContext(ctx).WithFields(logrus.Fields{"body": body}).Error(genHttpText(ctx, name, nil, "反序列化异常"))
		return err
	}
	return response.HttpSuccess(ctx)
}
func DealHttpResponse(ctx context.Context, name string, response *resty.Response, err error) (string, error) {
	if err != nil {
		logrus.WithContext(ctx).WithFields(logrus.Fields{"err": err}).Error(genHttpText(ctx, name, nil, "请求异常"))
		return "", errors.Errorf(genHttpText(ctx, name, err, "请求异常"))
	}
	if response == nil {
		logrus.WithContext(ctx).WithFields(logrus.Fields{}).Error(genHttpText(ctx, name, nil, "响应为空"))
		return "", errors.Errorf(genHttpText(ctx, name, nil, "响应为空"))
	}
	statusCode := response.StatusCode()
	body := response.String()
	logrus.WithContext(ctx).WithFields(logrus.Fields{"statusCode": statusCode, "body": len(body)}).Info(genHttpText(ctx, name, nil, "响应"))
	if statusCode != http.StatusOK {
		logrus.WithContext(ctx).WithFields(logrus.Fields{"statusCode": statusCode}).Error(genHttpText(ctx, name, nil, "响应码失败"))
		return "", errors.Errorf(genHttpText(ctx, name, statusCode, "响应码失败"))
	}
	return body, nil
}
func genHttpText(ctx context.Context, name string, value interface{}, texts ...string) string {
	var str string
	if len(texts) == 0 {
		str = name
	} else {
		str = fmt.Sprintf("%s，%s", name, strings.Join(texts, "，"))
	}
	if value != nil {
		str = fmt.Sprintf("%s: %+v", str, value)
	}
	return str
}

func GetIp() string {
	return ip
}
func flushHttpGetIp(ctx context.Context, pool *SingleGoPool) {
	defer Defer(func(err interface{}, stack string) {
		logrus.WithContext(ctx).WithFields(logrus.Fields{"err": err, "stack": stack}).Error("HttpGetIp，退出")
	})

	for {
		ctx := ResetLogId(ctx)
		object := HttpGetIp(ctx)
		object = strings.TrimSpace(object)
		if object != "" {
			ip = object
		}
		Sleep(ctx, time.Hour)
		if CtxDone(ctx) {
			return
		}
	}
}
func HttpGetIp(ctx context.Context) string {
	response, err := GetHttpSpiderRequest(ctx).Get("https://ifconfig.co/ip")
	body, _ := DealHttpResponse(ctx, "HttpGetIp", response, err)
	return body
}

func GetHttpSpiderRequest(ctx context.Context) *resty.Request {
	return GetHttpClientSpider().R().SetContext(ctx)
}
func GetHttpRequest(ctx context.Context) *resty.Request {
	return GetHttpClient().R().SetContext(ctx)
}
func GetHttpClient() *resty.Client {
	httpClientOnce.Do(func() {
		httpClient = CreateHttpClient(TimeoutDefault, 0, nil, nil, true)
	})
	return httpClient
}
func GetHttpClientSpider() *resty.Client {
	httpClientSpiderOnce.Do(func() {
		httpClientSpider = CreateHttpClient(TimeoutDefault, 0, SpiderSleepsDefault, nil, true)
	})
	return httpClientSpider
}
func CreateHttpClient(timeout time.Duration, try int, sleeps []time.Duration, header map[string]string, skipTls bool) *resty.Client {
	client := resty.New()
	if timeout > 0 {
		client = client.SetTimeout(timeout)
	}
	if try < len(sleeps)+1 {
		try = len(sleeps) + 1
	}
	if try > 1 {
		client = client.SetRetryCount(try - 1)
		client = client.SetRetryWaitTime(GetSleepTime(sleeps, 0))
		client = client.SetRetryMaxWaitTime(GetSleepTime(sleeps, len(sleeps)))
		client = client.AddRetryCondition(func(response *resty.Response, err error) bool {
			var ctx context.Context
			if response != nil && response.Request != nil {
				ctx = response.Request.Context()
			}
			if ctx == nil {
				ctx = GenCtx()
			}
			ctx = SetLogId(ctx)
			if CtxDone(ctx) {
				return false
			}
			if errors.Is(err, HttpBan) {
				logrus.WithContext(ctx).WithFields(logrus.Fields{}).Warn("HTTP请求异常，请求封禁")
				return false
			}
			if err != nil {
				logrus.WithContext(ctx).WithFields(logrus.Fields{"err": err}).Warn("HTTP请求异常，重试请求")
				return true
			}
			var statusCode int
			if response != nil {
				statusCode = response.StatusCode()
			}
			if statusCode == 404 {
				logrus.WithContext(ctx).WithFields(logrus.Fields{"statusCode": statusCode}).Warn("HTTP请求异常，请求404")
				return false
			}
			if 400 <= statusCode && statusCode < 500 && statusCode != 404 {
				logrus.WithContext(ctx).WithFields(logrus.Fields{"statusCode": statusCode}).Warn("HTTP请求异常，请求封禁")
				if response.Request != nil {
					setHttpBan(ctx, response.Request.URL, SleepDefault)
				}
				return false
			}
			if 500 <= statusCode {
				logrus.WithContext(ctx).WithFields(logrus.Fields{"statusCode": statusCode}).Warn("HTTP请求异常，重试请求")
				return true
			}
			return false
		})
		client = client.SetRetryAfter(func(client *resty.Client, response *resty.Response) (time.Duration, error) {
			var ctx context.Context
			if response != nil && response.Request != nil {
				ctx = response.Request.Context()
			}
			if ctx == nil {
				ctx = GenCtx()
			}
			ctx = SetLogId(ctx)
			if CtxDone(ctx) {
				logrus.WithContext(ctx).WithFields(logrus.Fields{}).Error("HTTP请求异常，重试超时")
				return 0, errors.Errorf("HTTP请求异常，重试超时")
			}
			var attempt int
			if response != nil && response.Request != nil {
				attempt = response.Request.Attempt
			}
			if try <= attempt {
				logrus.WithContext(ctx).WithFields(logrus.Fields{"attempt": attempt}).Error("HTTP请求异常，重试超限")
				return 0, errors.Errorf("HTTP请求异常，重试超限")
			}
			wareSleep := GetSleepTime(sleeps, attempt-1)
			wareSleep = WareDuration(wareSleep)
			logrus.WithContext(ctx).WithFields(logrus.Fields{"attempt": attempt, "wareSleep": wareSleep}).Warn("HTTP请求异常，休眠重试")
			return wareSleep, nil
		})
	}
	client = client.OnBeforeRequest(func(client *resty.Client, request *resty.Request) error {
		var ctx context.Context
		if request != nil {
			ctx = request.Context()
		}
		if ctx == nil {
			ctx = GenCtx()
		}
		ctx = SetLogId(ctx)
		address := request.URL
		if getHttpBan(ctx, address) {
			logrus.WithContext(ctx).WithFields(logrus.Fields{"address": address}).Warn("HTTP请求异常，请求封禁")
			return HttpBan
		}
		return nil
	})
	if header == nil {
		header = make(map[string]string, 1)
	}
	if header[UserAgentKey] == "" {
		header[UserAgentKey] = UserAgentDefault
	}
	for key := range header {
		client = client.SetHeader(key, header[key])
	}
	client = client.SetTLSClientConfig(&tls.Config{InsecureSkipVerify: skipTls})
	return client
}
func GetSleepTime(sleeps []time.Duration, index int) time.Duration {
	if len(sleeps) == 0 {
		return 1
	}
	sleep := sleeps[len(sleeps)-1]
	if index < len(sleeps) {
		sleep = sleeps[index]
	}
	if sleep <= 0 {
		sleep = 1
	}
	return sleep
}

func ParseCurl(ctx context.Context, curl string) (*model.HttpRequestParam, error) {
	var param model.HttpRequestParam
	param.Header = make(map[string]string)
	lines := strings.Split(curl, "\n")
	for i := range lines {
		line := lines[i]
		line = strings.TrimSpace(line)
		if strings.HasSuffix(line, "\\") {
			line = line[:len(line)-1]
		}
		if strings.HasPrefix(line, "curl") {
			line = line[4:]
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "'") || strings.HasPrefix(line, "\"") {
				line = line[1:]
			}
			if strings.HasSuffix(line, "'") || strings.HasSuffix(line, "\"") {
				line = line[:len(line)-1]
			}
			param.Url = line
			continue
		}
		if strings.HasPrefix(line, "-H") {
			line = line[2:]
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "'") || strings.HasPrefix(line, "\"") {
				line = line[1:]
			}
			if strings.HasSuffix(line, "'") || strings.HasSuffix(line, "\"") {
				line = line[:len(line)-1]
			}
			ss := strings.Split(line, ":")
			if len(ss) < 2 {
				continue
			}
			key := ss[0]
			key = strings.TrimSpace(key)
			value := ss[1]
			value = strings.TrimSpace(value)
			param.Header[key] = value
			continue
		}
		if strings.HasPrefix(line, "--data-raw") {
			line = line[10:]
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "'") || strings.HasPrefix(line, "\"") {
				line = line[1:]
			}
			if strings.HasSuffix(line, "'") || strings.HasSuffix(line, "\"") {
				line = line[:len(line)-1]
			}
			param.Body = line
			continue
		}
	}
	return &param, nil
}
