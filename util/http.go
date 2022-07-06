package util

import (
	"context"
	"crypto/tls"
	"fmt"
	"github.com/cellargalaxy/go_common/model"
	"github.com/gin-gonic/gin"
	"github.com/go-resty/resty/v2"
	"github.com/sirupsen/logrus"
	"net/http"
	"strings"
	"time"
)

const TokenKey = "Authorization"
const BearerKey = "Bearer"
const ClaimsKey = "claims"

var HttpClientNotReTry *resty.Client
var ip string

func initHttp(ctx context.Context) {
	HttpClientNotReTry = CreateNotReTryHttpClient(time.Second * 5)
	flushHttpIpAsync(ctx)
}

func CreateErrResponse(message string) map[string]interface{} {
	return createResponse(HttpFailCode, message, nil)
}

func CreateResponse(data interface{}, err error) map[string]interface{} {
	if err == nil {
		return createResponse(HttpSuccessCode, "", data)
	} else {
		return createResponse(HttpFailCode, err.Error(), data)
	}
}

func createResponse(code int, msg string, data interface{}) map[string]interface{} {
	return gin.H{"code": code, "msg": msg, "data": data}
}

func Ping(c *gin.Context) {
	logrus.WithContext(c).WithFields(logrus.Fields{"claims": GetClaims(c)}).Info("Ping")
	c.JSON(http.StatusOK, CreateResponse(model.PingResponse{Timestamp: time.Now().Unix(), ServerName: GetServerName("")}, nil))
}

func GetClaims(ctx context.Context) *model.Claims {
	object := GetCtxValue(ctx, ClaimsKey)
	claims, _ := object.(*model.Claims)
	return claims
}
func SetClaims(ctx context.Context, claims *model.Claims) context.Context {
	if claims == nil {
		return ctx
	}
	return SetCtxValue(ctx, ClaimsKey, claims)
}

func initGinLogId(c *gin.Context) {
	logId := GetLogId(c)
	if logId <= 0 {
		logId = String2Int64(c.Request.Header.Get(LogIdKey))
	}
	if logId <= 0 {
		logId = GenLogId()
	}
	c.Set(LogIdKey, logId)
	c.Header(LogIdKey, GetLogIdString(c))
}

func HttpClaims(c *gin.Context, secret string) {
	defer func() {
		initGinLogId(c)
		c.Next()
	}()

	token := c.Request.Header.Get(TokenKey)
	tokens := strings.SplitN(token, " ", 2)
	if len(tokens) != 2 || tokens[0] != BearerKey {
		return
	}
	var claims model.Claims
	jwtToken, err := ParseJWT(c, tokens[1], secret, &claims)
	c.Set(LogIdKey, claims.LogId)
	if err != nil {
		return
	}
	if jwtToken == nil {
		return
	}
	if !jwtToken.Valid {
		return
	}
	c.Set(ClaimsKey, &claims)
}

func HttpValidate(c *gin.Context, secret string) {
	token := c.Request.Header.Get(TokenKey)
	tokens := strings.SplitN(token, " ", 2)
	if len(tokens) != 2 || tokens[0] != BearerKey {
		initGinLogId(c)
		c.Abort()
		c.JSON(http.StatusOK, CreateErrResponse("Authorization非法"))
		return
	}
	var claims model.Claims
	jwtToken, err := ParseJWT(c, tokens[1], secret, &claims)
	c.Set(LogIdKey, claims.LogId)
	if err != nil {
		initGinLogId(c)
		c.Abort()
		c.JSON(http.StatusOK, CreateErrResponse(err.Error()))
		return
	}
	if jwtToken == nil {
		initGinLogId(c)
		c.Abort()
		c.JSON(http.StatusOK, CreateErrResponse("jwtToken为空"))
		return
	}
	if !jwtToken.Valid {
		initGinLogId(c)
		c.Abort()
		c.JSON(http.StatusOK, CreateErrResponse("jwtToken非法"))
		return
	}
	c.Set(ClaimsKey, &claims)

	if claims.ReqId == "" {
		initGinLogId(c)
		c.Abort()
		c.JSON(http.StatusOK, CreateErrResponse("ReqId为空"))
		return
	}
	expiresAt := time.Unix(claims.ExpiresAt, 0)
	duration := expiresAt.Sub(time.Now())
	if duration.Nanoseconds() <= 0 {
		initGinLogId(c)
		c.Abort()
		c.JSON(http.StatusOK, CreateErrResponse("jwtToken过期"))
		return
	}
	if existRequestId(claims.ReqId, duration) {
		initGinLogId(c)
		c.Abort()
		c.JSON(http.StatusOK, createResponse(HttpReRequestCode, "请求重放非法", nil))
		return
	}
	initGinLogId(c)
	c.Next()
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

func GetIp() string {
	return ip
}

func flushHttpIpAsync(ctx context.Context) {
	go func() {
		defer Defer(ctx, func(ctx context.Context, err interface{}, stack string) {
			logrus.WithContext(ctx).WithFields(logrus.Fields{"err": err, "stack": stack}).Warn("异步刷新IP，退出")
			flushHttpIpAsync(ctx)
		})

		for {
			FlushHttpIp(ctx)
			time.Sleep(time.Hour)
		}
	}()
}

func FlushHttpIp(ctx context.Context) {
	for i := 0; i < 10; i++ {
		object := GetHttpIp(ctx)
		object = strings.TrimSpace(object)
		if object == "" {
			time.Sleep(time.Second)
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
	response, err := HttpClientNotReTry.R().SetContext(ctx).Get(url)
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

func CreateNotReTryHttpClient(timeout time.Duration) *resty.Client {
	return CreateHttpClient(timeout, 0, 0, 0, nil, true)
}

func CreateHttpClient(timeout, sleep, maxSleep time.Duration, retry int, header map[string]string, skipTls bool) *resty.Client {
	client := resty.New()
	if timeout > 0 {
		client = client.SetTimeout(timeout)
	}
	if retry > 0 {
		client = client.SetRetryCount(retry)
		if sleep > 0 {
			client = client.SetRetryWaitTime(sleep)
		}
		if maxSleep > 0 {
			client = client.SetRetryMaxWaitTime(maxSleep)
		}
		client = client.AddRetryCondition(func(response *resty.Response, err error) bool {
			var ctx context.Context
			if response != nil && response.Request != nil {
				ctx = response.Request.Context()
			}
			if ctx == nil || GetLogId(ctx) <= 0 {
				ctx = GenCtx()
			}
			var statusCode int
			if response != nil {
				statusCode = response.StatusCode()
			}
			isRetry := statusCode >= 500 || err != nil
			if isRetry {
				logrus.WithContext(ctx).WithFields(logrus.Fields{"statusCode": statusCode, "err": err}).Warn("HTTP请求异常，进行重试")
			}
			return isRetry
		})
		client = client.SetRetryAfter(func(client *resty.Client, response *resty.Response) (time.Duration, error) {
			var ctx context.Context
			if response != nil && response.Request != nil {
				ctx = response.Request.Context()
			}
			if ctx == nil || GetLogId(ctx) <= 0 {
				ctx = GenCtx()
			}
			var attempt int
			if response != nil && response.Request != nil {
				attempt = response.Request.Attempt
			}
			if attempt > retry {
				logrus.WithContext(ctx).WithFields(logrus.Fields{"attempt": attempt}).Error("HTTP请求异常，超过最大重试次数")
				return 0, fmt.Errorf("HTTP请求异常，超过最大重试次数")
			}
			wareSleep := sleep
			for i := 0; i < attempt-1; i++ {
				wareSleep *= 10
			}
			wareSleep = WareDuration(wareSleep)
			logrus.WithContext(ctx).WithFields(logrus.Fields{"attempt": attempt, "wareSleep": wareSleep}).Warn("HTTP请求异常，休眠重试")
			return wareSleep, nil
		})
	}
	for key := range header {
		client = client.SetHeader(key, header[key])
	}
	if skipTls {
		client = client.SetTLSClientConfig(&tls.Config{InsecureSkipVerify: true})
	}
	return client
}
