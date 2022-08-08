package util

import (
	"context"
	"crypto/tls"
	"fmt"
	"github.com/cellargalaxy/go_common/model"
	"github.com/go-resty/resty/v2"
	"github.com/sirupsen/logrus"
	"net/http"
	"strings"
	"sync"
	"time"
)

const (
	TimeoutDefault   = time.Second * 3
	SleepDefault     = time.Second * 3
	RetryDefault     = 3
	UserAgentDefault = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/83.0.4103.116 Safari/537.36"
)

var SpiderSleepsDefault []time.Duration
var httpClient *resty.Client
var httpClientOnce sync.Once
var httpClientSpider *resty.Client
var httpClientSpiderOnce sync.Once
var ip string

func initHttp(ctx context.Context) {
	SpiderSleepsDefault = []time.Duration{time.Millisecond, time.Second * 2, time.Minute, time.Minute, time.Minute * 5, time.Minute * 15}
	flushHttpIpAsync(ctx)
}

func DealHttpApiRequest(ctx context.Context, name string, response *resty.Response, err error) (string, error) {
	if err != nil {
		logrus.WithContext(ctx).WithFields(logrus.Fields{"err": err}).Error(getDealHttpApiRequest(ctx, name, nil, "请求异常"))
		return "", fmt.Errorf(getDealHttpApiRequest(ctx, name, err, "请求异常"))
	}
	if response == nil {
		logrus.WithContext(ctx).WithFields(logrus.Fields{}).Error(getDealHttpApiRequest(ctx, name, nil, "响应为空"))
		return "", fmt.Errorf(getDealHttpApiRequest(ctx, name, nil, "响应为空"))
	}
	statusCode := response.StatusCode()
	body := response.String()
	logrus.WithContext(ctx).WithFields(logrus.Fields{"statusCode": statusCode, "body": len(body)}).Info(getDealHttpApiRequest(ctx, name, nil, "响应"))
	if statusCode != http.StatusOK {
		logrus.WithContext(ctx).WithFields(logrus.Fields{"statusCode": statusCode}).Error(getDealHttpApiRequest(ctx, name, nil, "响应码失败"))
		return "", fmt.Errorf(getDealHttpApiRequest(ctx, name, statusCode, "响应码失败"))
	}
	return body, nil
}
func getDealHttpApiRequest(ctx context.Context, name string, value interface{}, texts ...string) string {
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

func flushHttpIpAsync(ctx context.Context) {
	go func() {
		defer Defer(func(err interface{}, stack string) {
			logrus.WithContext(ctx).WithFields(logrus.Fields{"err": err, "stack": stack}).Warn("异步刷新IP，退出")
			flushHttpIpAsync(ctx)
		})

		for {
			FlushHttpIp(ctx)
			Sleep(ctx, time.Hour)
		}
	}()
}

func FlushHttpIp(ctx context.Context) {
	for i := 0; i < 10; i++ {
		object := GetHttpIp(ctx)
		object = strings.TrimSpace(object)
		if object == "" {
			Sleep(ctx, time.Second)
			continue
		}
		ip = object
		return
	}
}

func GetHttpIp(ctx context.Context) string {
	return HttpGet(ctx, "https://ifconfig.co/ip")
}

func HttpGet(ctx context.Context, url string) string {
	response, err := GetHttpClient().R().SetContext(ctx).Get(url)
	if err != nil {
		logrus.WithContext(ctx).WithFields(logrus.Fields{"url": url, "err": err}).Error("HttpGet，请求异常")
		return ""
	}
	if response == nil {
		logrus.WithContext(ctx).WithFields(logrus.Fields{"url": url}).Error("HttpGet，响应为空")
		return ""
	}
	statusCode := response.StatusCode()
	body := response.String()
	logrus.WithContext(ctx).WithFields(logrus.Fields{"statusCode": statusCode, "len(body)": len(body)}).Info("HttpGet，响应")
	if statusCode != http.StatusOK {
		return ""
	}
	return body
}

func GetHttpClient() *resty.Client {
	httpClientOnce.Do(func() {
		httpClient = CreateTimeoutHttpClient(TimeoutDefault)
	})
	return httpClient
}

func GetHttpClientSpider() *resty.Client {
	httpClientSpiderOnce.Do(func() {
		httpClientSpider = CreateHttpClient(TimeoutDefault, RetryDefault, SpiderSleepsDefault, nil, true)
	})
	return httpClientSpider
}

func CreateTimeoutHttpClient(timeout time.Duration) *resty.Client {
	return CreateHttpClient(timeout, 0, nil, nil, true)
}

func CreateHttpClient(timeout time.Duration, retry int, sleeps []time.Duration, header map[string]string, skipTls bool) *resty.Client {
	client := resty.New()
	if timeout > 0 {
		client = client.SetTimeout(timeout)
	}
	if retry < len(sleeps) {
		retry = len(sleeps)
	}
	if retry > 0 {
		client = client.SetRetryCount(retry)
		sleep := SleepDefault
		if len(sleeps) > 0 {
			sleep = sleeps[0]
		}
		client = client.SetRetryWaitTime(sleep)
		client = client.AddRetryCondition(func(response *resty.Response, err error) bool {
			var ctx context.Context
			if response != nil && response.Request != nil {
				ctx = response.Request.Context()
			}
			if ctx == nil || GetLogId(ctx) <= 0 {
				ctx = GenCtx()
			}
			if CtxDone(ctx) {
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
			if statusCode >= 500 {
				logrus.WithContext(ctx).WithFields(logrus.Fields{"err": err}).Warn("HTTP请求异常，重试请求")
				return true
			}
			return false
		})
		client = client.SetRetryAfter(func(client *resty.Client, response *resty.Response) (time.Duration, error) {
			var ctx context.Context
			if response != nil && response.Request != nil {
				ctx = response.Request.Context()
			}
			if ctx == nil || GetLogId(ctx) <= 0 {
				ctx = GenCtx()
			}
			if CtxDone(ctx) {
				logrus.WithContext(ctx).WithFields(logrus.Fields{}).Error("HTTP请求异常，重试超时")
				return 0, fmt.Errorf("HTTP请求异常，重试超时")
			}
			var attempt int
			if response != nil && response.Request != nil {
				attempt = response.Request.Attempt
			}
			if retry < attempt {
				logrus.WithContext(ctx).WithFields(logrus.Fields{"attempt": attempt}).Error("HTTP请求异常，重试超限")
				return 0, fmt.Errorf("HTTP请求异常，重试超限")
			}
			wareSleep := sleep
			if 0 <= attempt && attempt < len(sleeps) {
				wareSleep = sleeps[attempt]
			} else if len(sleeps) > 0 {
				wareSleep = sleeps[len(sleeps)-1]
			}
			wareSleep = WareDuration(wareSleep)
			logrus.WithContext(ctx).WithFields(logrus.Fields{"attempt": attempt, "wareSleep": wareSleep}).Warn("HTTP请求异常，休眠重试")
			return wareSleep, nil
		})
	}
	if header == nil {
		header = make(map[string]string, 1)
	}
	if header["User-Agent"] == "" {
		header["User-Agent"] = UserAgentDefault
	}
	for key := range header {
		client = client.SetHeader(key, header[key])
	}
	client = client.SetTLSClientConfig(&tls.Config{InsecureSkipVerify: skipTls})
	return client
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
