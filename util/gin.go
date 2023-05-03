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

func NewHttpRespByErr(data interface{}, err error) model.HttpResponse {
	var msg string
	if err != nil {
		msg = err.Error()
	}
	return NewHttpRespByMsg(data, msg)
}
func NewHttpRespByMsg(data interface{}, msg string) model.HttpResponse {
	if msg == "" {
		return NewHttpResp(model.HttpSuccessCode, "", data)
	} else {
		return NewHttpResp(model.HttpFailCode, msg, data)
	}
}
func NewHttpResp(code int, msg string, data interface{}) model.HttpResponse {
	return model.HttpResponse{Code: code, Msg: msg, Data: data}
}

func Ping(c *gin.Context) {
	logrus.WithContext(c).WithFields(logrus.Fields{"claims": GetClaims(c)}).Info("Ping")
	c.JSON(http.StatusOK, NewHttpRespByErr(model.PingResponse{Timestamp: time.Now().Unix(), ServerName: GetServerName()}, nil))
}

func GetClaims(ctx context.Context) *model.Claims {
	return GetCtxValue[*model.Claims](ctx, ClaimsKey)
}
func SetClaims(ctx context.Context, claims *model.Claims) context.Context {
	if claims == nil {
		return ctx
	}
	return SetCtxValue(ctx, ClaimsKey, claims)
}

func setGinLogId(c *gin.Context) {
	logId := GetLogId(c)
	if logId <= 0 {
		logId = GenLogId()
	}
	c.Set(LogIdKey, logId)
	c.Header(LogIdKey, Int642String(logId))
}
func ClaimsGin(c *gin.Context, secret string) {
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
	jwtToken, err := DeJwt(c, token, secret, &claims)
	if claims.LogId > 0 {
		c.Set(LogIdKey, claims.LogId)
	}
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
func ValidateGin(c *gin.Context, secret string) {
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
		c.JSON(http.StatusOK, NewHttpRespByMsg(nil, "Authorization非法"))
		return
	}
	var claims model.Claims
	jwtToken, err := DeJwt(c, token, secret, &claims)
	if err != nil {
		c.Abort()
		c.JSON(http.StatusOK, NewHttpRespByErr(nil, err))
		return
	}
	if jwtToken == nil {
		c.Abort()
		c.JSON(http.StatusOK, NewHttpRespByMsg(nil, "jwtToken为空"))
		return
	}
	if !jwtToken.Valid {
		c.Abort()
		c.JSON(http.StatusOK, NewHttpRespByMsg(nil, "jwtToken非法"))
		return
	}

	expiresAt := time.Unix(claims.ExpiresAt, 0)
	duration := expiresAt.Sub(time.Now())
	if duration.Nanoseconds() <= 0 {
		c.Abort()
		c.JSON(http.StatusOK, NewHttpRespByMsg(nil, "jwtToken过期"))
		return
	}
	if claims.ReqId != "" {
		if existReqId(c, claims.ReqId, duration) {
			c.Abort()
			c.JSON(http.StatusOK, NewHttpResp(model.HttpReRequestCode, "请求非法重放", nil))
			return
		}
	}
	if claims.Uri != "" {
		uri := c.Request.RequestURI
		uri = strings.Split(uri, "#")[0]
		uri = strings.Split(uri, "?")[0]
		if claims.Uri != uri {
			c.Abort()
			c.JSON(http.StatusOK, NewHttpResp(model.HttpIllegalUriCode, "请求非法uri", nil))
			return
		}
	}
	c.Next()
}
