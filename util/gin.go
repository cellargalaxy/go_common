package util

import (
	"context"
	"github.com/cellargalaxy/go_common/model"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"net/http"
	"strings"
	"time"
)

const (
	AuthorizationKey = "Authorization"
	BearerKey        = "Bearer"
	ClaimsKey        = "claims"
)

func CreateResponseByErr(err error) map[string]interface{} {
	return CreateResponse(nil, err)
}

func CreateFailResponse(message string) map[string]interface{} {
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
	c.JSON(http.StatusOK, CreateResponse(model.PingResponse{Timestamp: time.Now().Unix(), ServerName: GetServerName()}, nil))
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

func setGinLogId(c *gin.Context) {
	var logId int64
	object, ok := c.Get(LogIdKey)
	if object != nil && ok {
		logId, _ = object.(int64)
	}
	if logId <= 0 {
		logId = GenLogId()
	}
	c.Set(LogIdKey, logId)
	c.Header(LogIdKey, Int642String(logId))
}

func ClaimsHttp(c *gin.Context, secret string) {
	setGinLogId(c)
	defer c.Next()

	var token string
	authorization := c.Request.Header.Get(AuthorizationKey)
	authorizations := strings.SplitN(authorization, " ", 2)
	if len(authorizations) == 2 && authorizations[0] == BearerKey {
		token = authorizations[1]
	}
	if token == "" {
		token = c.Query(AuthorizationKey)
	}
	if token == "" {
		return
	}
	var claims model.Claims
	jwtToken, err := ParseJWT(c, token, secret, &claims)
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

func ValidateHttp(c *gin.Context, secret string) {
	setGinLogId(c)

	var token string
	authorization := c.Request.Header.Get(AuthorizationKey)
	authorizations := strings.SplitN(authorization, " ", 2)
	if len(authorizations) == 2 && authorizations[0] == BearerKey {
		token = authorizations[1]
	}
	if token == "" {
		token = c.Query(AuthorizationKey)
	}
	if token == "" {
		c.Abort()
		c.JSON(http.StatusOK, CreateFailResponse("Authorization非法"))
		return
	}
	var claims model.Claims
	jwtToken, err := ParseJWT(c, token, secret, &claims)
	if err != nil {
		c.Abort()
		c.JSON(http.StatusOK, CreateResponseByErr(err))
		return
	}
	if jwtToken == nil {
		c.Abort()
		c.JSON(http.StatusOK, CreateFailResponse("jwtToken为空"))
		return
	}
	if !jwtToken.Valid {
		c.Abort()
		c.JSON(http.StatusOK, CreateFailResponse("jwtToken非法"))
		return
	}

	expiresAt := time.Unix(claims.ExpiresAt, 0)
	duration := expiresAt.Sub(time.Now())
	if duration.Nanoseconds() <= 0 {
		c.Abort()
		c.JSON(http.StatusOK, CreateFailResponse("jwtToken过期"))
		return
	}
	if claims.ReqId != "" {
		if existReqId(claims.ReqId, duration) {
			c.Abort()
			c.JSON(http.StatusOK, createResponse(HttpReRequestCode, "请求非法重放", nil))
			return
		}
	}
	if claims.Uri != "" {
		if claims.Uri != c.Request.RequestURI {
			c.Abort()
			c.JSON(http.StatusOK, createResponse(HttpReRequestCode, "请求非法uri", nil))
			return
		}
	}
	c.Next()
}
