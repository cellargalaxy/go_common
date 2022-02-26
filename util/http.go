package util

import (
	"context"
	"github.com/cellargalaxy/go_common/consd"
	"github.com/cellargalaxy/go_common/model"
	"github.com/gin-gonic/gin"
	"github.com/patrickmn/go-cache"
	"github.com/sirupsen/logrus"
	"net/http"
	"strings"
	"time"
)

const TokenKey = "Authorization"
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
	return createResponse(consd.HttpFailCode, message, nil)
}

func CreateResponse(data interface{}, err error) map[string]interface{} {
	if err == nil {
		return createResponse(consd.HttpSuccessCode, "", data)
	} else {
		return createResponse(consd.HttpFailCode, err.Error(), data)
	}
}

func createResponse(code int, msg string, data interface{}) map[string]interface{} {
	return gin.H{"code": code, "msg": msg, "data": data}
}

func Ping(context *gin.Context) {
	context.JSON(http.StatusOK, gin.H{"code": consd.HttpSuccessCode, "msg": "pong", "data": map[string]interface{}{"ts": time.Now().Unix(), serverNameEnvKey: GetServerName("")}})
}

//token检查
func HttpValidate(c *gin.Context, validateHandler model.HttpValidateInter) {
	token := c.Request.Header.Get(TokenKey)
	logrus.WithContext(c).WithFields(logrus.Fields{"token": token}).Info("解析token")
	tokens := strings.SplitN(token, " ", 2)
	if len(tokens) != 2 || tokens[0] != "Bearer" {
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
			c.JSON(http.StatusOK, createResponse(consd.HttpReRequestCode, "JWT 请求重放非法", nil))
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
