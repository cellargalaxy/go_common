package util

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"strings"
	"sync/atomic"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

const (
	TimeoutDefault   = time.Second * 3
	SleepDefault     = time.Second * 3
	TryDefault       = 3
	UserAgentKey     = "User-Agent"
	UserAgentDefault = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/83.0.4103.116 Safari/537.36"
)

var httpClient = CreateHttpClient(TimeoutDefault, []time.Duration{time.Second, time.Second * 2, time.Second * 2}, nil, true)
var ip atomic.Value

func initHttp() {
	var err error
	ctx := GenCtx()
	_, err = NewDaemonSingleGoPool(ctx, "HttpGetIp", time.Hour, flushIP)
	if err != nil {
		panic(err)
	}
}

func CreateHttpClient(timeout time.Duration, sleeps []time.Duration, header map[string]string, skipTls bool) *resty.Client {
	var GetSleepTime = func(sleeps []time.Duration, index int) time.Duration {
		if len(sleeps) == 0 {
			return 1
		}
		if index < 0 {
			index = 0
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
	var HttpBan = errors.Errorf("HTTP请求封禁")

	client := resty.New()
	if timeout > 0 {
		client = client.SetTimeout(timeout)
	}
	if len(sleeps) > 1 {

		client = client.SetRetryCount(len(sleeps) - 1)
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
			if statusCode == http.StatusNotFound {
				logrus.WithContext(ctx).WithFields(logrus.Fields{"statusCode": statusCode}).Warn("HTTP请求异常，请求404")
				return false
			}
			if http.StatusBadRequest <= statusCode && statusCode < http.StatusInternalServerError {
				logrus.WithContext(ctx).WithFields(logrus.Fields{"statusCode": statusCode}).Warn("HTTP请求异常，请求封禁")
				if response.Request != nil {
					SetHttpBan(ctx, response.Request.URL, SleepDefault)
				}
				return false
			}
			if http.StatusInternalServerError <= statusCode {
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
			if len(sleeps) <= attempt {
				logrus.WithContext(ctx).WithFields(logrus.Fields{"attempt": attempt}).Error("HTTP请求异常，重试超限")
				return 0, errors.Errorf("HTTP请求异常，重试超限")
			}
			wareSleep := GetSleepTime(sleeps, attempt-1)
			wareSleep = WareNumber(wareSleep)
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
		if GetHttpBan(ctx, address) {
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

func NewHttpClientReq(ctx context.Context) *resty.Request {
	return httpClient.R().SetContext(ctx)
}
func DealHttpClientResp(ctx context.Context, name string, response *resty.Response, err error) (string, error) {
	var GenMsg = func(name string, value interface{}, texts ...string) string {
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

	if err != nil {
		logrus.WithContext(ctx).WithFields(logrus.Fields{"err": err}).Error(GenMsg(name, nil, "HTTP请求异常"))
		return "", errors.New(GenMsg(name, err, "HTTP请求异常"))
	}
	if response == nil {
		logrus.WithContext(ctx).WithFields(logrus.Fields{}).Error(GenMsg(name, nil, "HTTP响应为空"))
		return "", errors.New(GenMsg(name, nil, "HTTP响应为空"))
	}
	statusCode := response.StatusCode()
	body := response.String()
	logrus.WithContext(ctx).WithFields(logrus.Fields{"statusCode": statusCode, "body": len(body)}).Info(GenMsg(name, nil, "HTTP响应"))
	if statusCode != http.StatusOK {
		logrus.WithContext(ctx).WithFields(logrus.Fields{"statusCode": statusCode}).Error(GenMsg(name, nil, "HTTP响应码失败"))
		return "", errors.New(GenMsg(name, statusCode, "HTTP响应码失败"))
	}
	return body, nil
}

func LoadIP(ctx context.Context) string {
	response, err := NewHttpClientReq(ctx).Get("http://ipv4.duia.ro/")
	body, _ := DealHttpClientResp(ctx, "HttpGetIp", response, err)
	body = strings.TrimSpace(body)
	if body != "" {
		ip.Store(body)
	}
	return body
}
func GetIP() string {
	value, _ := ip.Load().(string)
	if value != "" {
		return value
	}
	ctx := GenCtx()
	return LoadIP(ctx)
}
func flushIP(ctx context.Context, pool *SingleGoPool) {
	defer Defer(func(err interface{}, stack string) {
		if err != nil {
			logrus.WithContext(ctx).WithFields(logrus.Fields{"err": err, "stack": stack}).Error("HttpGetIp，退出")
		}
	})

	for {
		ccc := ReSetLogId(ctx)
		LoadIP(ccc)
		Sleep(ccc, time.Hour)
		if CtxDone(ccc) {
			return
		}
	}
}

func ParseCurl(ctx context.Context, curl string) (url string, header map[string]string, body string, err error) {
	header = make(map[string]string)
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
			url = line
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
			//只能按首个冒号切分：HTTP头值本身常含冒号
			//(Referer: https://x.com、Authorization: Bearer a:b、X-Time: 10:20:30)，
			//用 Split 后取 ss[1] 会把值截断成 "https"、"10"，请求头随之失真
			ss := strings.SplitN(line, ":", 2)
			if len(ss) < 2 {
				continue
			}
			key := ss[0]
			key = strings.TrimSpace(key)
			value := ss[1]
			value = strings.TrimSpace(value)
			header[key] = value
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
			body = line
			continue
		}
	}
	return url, header, body, nil
}
func ExecCurl(ctx context.Context, name, method, url string, header map[string]string) (string, error) {
	method = strings.TrimSpace(method)
	method = strings.ToUpper(method)
	switch method {
	case "", "GET", "POST":
	default:
		logrus.WithContext(ctx).WithFields(logrus.Fields{"method": method}).Error(fmt.Sprintf("%s，CURL请求异常，非法方法", name))
		return "", errors.Errorf("CURL请求异常，非法方法")
	}
	url = strings.TrimSpace(url)
	if url == "" {
		logrus.WithContext(ctx).WithFields(logrus.Fields{}).Error(fmt.Sprintf("%s，CURL请求异常，链接为空", name))
		return "", errors.Errorf("CURL请求异常，链接为空")
	}
	if header == nil {
		header = make(map[string]string)
	}
	if header[UserAgentKey] == "" {
		header[UserAgentKey] = UserAgentDefault
	}

	filename := fmt.Sprintf("/tmp/%d", GenId())
	//文件由下面的 >> 重定向创建，只要命令执行过就会落盘。
	//此前只在成功路径末尾删除，命令失败、响应码非200、读取失败等分支都会把文件留在/tmp里，
	//实测打一次失败请求即残留一个0字节文件，长期运行会持续堆积，故统一用defer兜底删除。
	defer RemoveFile(ctx, filename)
	curls := make([]string, 0, len(header)+2)
	curls = append(curls, fmt.Sprintf(`curl -v '%s' \`, url))
	if method == "POST" {
		curls = append(curls, `  -X 'POST' \`)
	}
	for key, value := range header {
		curls = append(curls, fmt.Sprintf(`  -H '%s: %s' \`, key, value))
	}
	curls = append(curls, fmt.Sprintf(`  --compressed >> %s`, filename))
	curl := strings.Join(curls, "\n")
	logrus.WithContext(ctx).WithFields(logrus.Fields{"curl": curl}).Info(fmt.Sprintf("%s，CURL请求", name))

	lines, stderrLines, err := ExecCommand(ctx, curl)
	if err != nil {
		return "", err
	}
	if len(lines) == 0 {
		lines = stderrLines
	}

	var statusCode int
	for i := range lines {
		lines[i] = strings.TrimSpace(lines[i])
		if !strings.Contains(lines[i], "< HTTP/") {
			continue
		}
		list := strings.Split(lines[i], "HTTP/")
		if len(list) < 2 {
			continue
		}
		list = strings.Split(list[1], " ")
		if len(list) >= 2 {
			statusCode = Str2Int[int](list[1])
		}
		break
	}
	if statusCode != http.StatusOK {
		logrus.WithContext(ctx).WithFields(logrus.Fields{"statusCode": statusCode}).Error(fmt.Sprintf("%s，响应码失败", name))
		return "", errors.Errorf("%s，响应码失败: %d", name, statusCode)
	}

	fileInfo := GetFileInfo(ctx, filename)
	if fileInfo == nil {
		logrus.WithContext(ctx).WithFields(logrus.Fields{}).Error("CURL请求异常，文件为空")
		return "", errors.Errorf("CURL请求异常，文件为空")
	}
	data, err := ReadFile2Str(ctx, filename, "")
	if err != nil {
		return "", err
	}

	return data, nil
}
