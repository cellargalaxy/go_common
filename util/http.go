package util

import (
	"context"
	"crypto/tls"
	"fmt"
	"github.com/cellargalaxy/go_common/model"
	"github.com/gin-gonic/gin"
	"github.com/go-resty/resty/v2"
	"github.com/patrickmn/go-cache"
	"github.com/sirupsen/logrus"
	"net/http"
	"strings"
	"time"
)

const TokenKey = "Authorization"
const BearerKey = "Bearer"
const ClaimsKey = "claims"

var httpLocalCache *cache.Cache

func initHttp() {
	httpLocalCache = cache.New(time.Second, time.Second)
	if httpLocalCache == nil {
		panic("创建本地缓存对象为空")
	}
}

func existRequestId(requestId string, duration time.Duration) bool {
	_, ok := httpLocalCache.Get(requestId)
	httpLocalCache.Set(requestId, requestId, duration)
	return ok
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

func Ping(context *gin.Context) {
	context.JSON(http.StatusOK, gin.H{"code": HttpSuccessCode, "msg": "pong", "data": map[string]interface{}{"ts": time.Now().Unix(), serverNameEnvKey: GetServerName("")}})
}

//token检查
func HttpValidate(c *gin.Context, validateHandler model.HttpValidateInter) {
	token := c.Request.Header.Get(TokenKey)
	logrus.WithContext(c).WithFields(logrus.Fields{"token": token}).Info("解析token")
	tokens := strings.SplitN(token, " ", 2)
	if len(tokens) != 2 || tokens[0] != BearerKey {
		c.Abort()
		c.JSON(http.StatusOK, CreateErrResponse("Authorization非法"))
		return
	}
	secret := validateHandler.GetSecret(c)
	claims := validateHandler.CreateClaims(c)
	jwtToken, err := ParseJWT(c, tokens[1], secret, claims)
	if err != nil {
		c.Abort()
		c.JSON(http.StatusOK, CreateErrResponse(err.Error()))
		return
	}
	if jwtToken == nil {
		c.Abort()
		c.JSON(http.StatusOK, CreateErrResponse("JWT token为空"))
		return
	}
	if !jwtToken.Valid {
		c.Abort()
		c.JSON(http.StatusOK, CreateErrResponse("JWT token非法"))
		return
	}
	if !claims.AllowReRequest {
		if claims.RequestId == "" {
			c.Abort()
			c.JSON(http.StatusOK, CreateErrResponse("JWT request_id为空"))
			return
		}
		expiresAt := time.Unix(claims.ExpiresAt, 0)
		duration := expiresAt.Sub(time.Now())
		if duration.Nanoseconds() <= 0 {
			c.Abort()
			c.JSON(http.StatusOK, CreateErrResponse("JWT token过期"))
			return
		}
		if existRequestId(claims.RequestId, duration) {
			c.Abort()
			c.JSON(http.StatusOK, createResponse(HttpReRequestCode, "JWT 请求重放非法", nil))
			return
		}
	}
	c.Set(ClaimsKey, jwtToken.Claims)
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

func HttpGet(url string) string {
	response, err := httpClient.R().Get(url)
	if err != nil {
		return ""
	}
	if response == nil {
		return ""
	}
	statusCode := response.StatusCode()
	body := response.String()
	if statusCode != http.StatusOK {
		return ""
	}
	return body
}

func HttpGetIp() string {
	return HttpGet("https://ifconfig.co/ip")
}

func CreateNotTryHttpClient(timeout time.Duration) *resty.Client {
	return CreateHttpClient(timeout, 0, 0, 0, nil, true)
}

func CreateHttpClient(timeout, sleep, maxSleep time.Duration, retry int, header map[string]string, skipTls bool) *resty.Client {
	client := resty.New()
	if timeout > 0 {
		client = client.SetTimeout(timeout)
	}
	if retry > 0 {
		client = client.SetRetryCount(retry)
		client = client.SetRetryWaitTime(sleep)
		client = client.SetRetryMaxWaitTime(maxSleep)
		client = client.AddRetryCondition(func(response *resty.Response, err error) bool {
			var ctx context.Context
			if response != nil && response.Request != nil {
				ctx = response.Request.Context()
			}
			if ctx == nil || GetLogId(ctx) <= 0 {
				ctx = CreateLogCtx()
			}
			var statusCode int
			if response != nil {
				statusCode = response.StatusCode()
			}
			isRetry := statusCode != http.StatusOK || err != nil
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
				ctx = CreateLogCtx()
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
