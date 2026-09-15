package util

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/cellargalaxy/go_common/model"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt"
	"github.com/sirupsen/logrus"
)

const (
	AuthorizationKey = "Authorization"
	BearerKey        = "Bearer"
	ClaimsKey        = "claims"
)

const (
	PathPing   = "/api/ping"
	PathStatic = "/static"
	PathDebug  = "/debug"
	PathPprof  = "/pprof"
)

func NewHttpRespByErr(data any, err error) model.HttpResp {
	var msg string
	if err != nil {
		msg = err.Error()
	}
	return NewHttpRespByMsg(data, msg)
}
func NewHttpRespByMsg(data interface{}, msg string) model.HttpResp {
	if msg == "" {
		return NewHttpResp(http.StatusOK, "", data)
	}
	return NewHttpResp(http.StatusInternalServerError, msg, data)
}
func NewHttpResp(code int, msg string, data interface{}) model.HttpResp {
	return model.HttpResp{Code: code, Msg: msg, Data: data}
}

func NewPingData() model.PingData {
	return model.PingData{Ip: GetIP(), ServerName: GetServerName(), Timestamp: time.Now().Unix()}
}
func Ping(c *gin.Context) {
	ctx := c.Request.Context()

	logrus.WithContext(ctx).WithFields(logrus.Fields{"claims": GetClaims[any](ctx)}).Info("Ping")
	c.JSON(http.StatusOK, NewHttpRespByErr(NewPingData(), nil))
}

func GetClaims[Claim any](ctx context.Context) Claim {
	return GetCtxValue[Claim](ctx, ClaimsKey)
}
func SetClaims(ctx context.Context, claims any) context.Context {
	if IsNil(claims) {
		return ctx
	}
	return SetCtxValue(ctx, ClaimsKey, claims)
}

func setGinLogId(c *gin.Context, logId int64) {
	if logId <= 0 {
		logId = GenId()
	}
	c.Request = c.Request.WithContext(SetCtxValue(c.Request.Context(), LogIdKey, logId))
	c.Header(LogIdKey, Int2Str(logId))
}

type Claims interface {
	jwt.Claims
	GetExpiresAt() int64
	GetLogId() int64
	GetReqId() int64
	GetUri() string
}

func ValidateGin(c *gin.Context, secret string, claims Claims) {
	setGinLogId(c, GetLogId(c.Request.Context()))

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
		c.JSON(http.StatusOK, NewHttpResp(http.StatusUnauthorized, "Authorization非法", nil))
		return
	}
	jwtToken, err := DeJwt(c.Request.Context(), token, secret, claims)
	if err != nil {
		c.Abort()
		c.JSON(http.StatusOK, NewHttpRespByErr(nil, err))
		return
	}
	if jwtToken == nil {
		c.Abort()
		c.JSON(http.StatusOK, NewHttpResp(http.StatusUnauthorized, "jwtToken为空", nil))
		return
	}
	if !jwtToken.Valid {
		c.Abort()
		c.JSON(http.StatusOK, NewHttpResp(http.StatusUnauthorized, "jwtToken非法", nil))
		return
	}

	if claims.GetLogId() > 0 {
		setGinLogId(c, claims.GetLogId())
	}

	expiresAt := time.Unix(claims.GetExpiresAt(), 0)
	duration := expiresAt.Sub(time.Now())
	if duration.Nanoseconds() <= 0 {
		c.Abort()
		c.JSON(http.StatusOK, NewHttpRespByMsg(nil, "jwtToken过期"))
		return
	}
	if claims.GetReqId() > 0 {
		if !TryLockReqId(c.Request.Context(), claims.GetReqId(), duration) {
			c.Abort()
			c.JSON(http.StatusOK, NewHttpResp(http.StatusConflict, "请求非法重放", nil))
			return
		}
	}
	if claims.GetUri() != "" {
		uri := c.Request.RequestURI
		uri = strings.Split(uri, "#")[0]
		uri = strings.Split(uri, "?")[0]
		if claims.GetUri() != uri {
			c.Abort()
			c.JSON(http.StatusOK, NewHttpResp(http.StatusBadRequest, "请求非法uri", nil))
			return
		}
	}
	c.Request = c.Request.WithContext(SetClaims(c.Request.Context(), claims))
	c.Next()
}

func NewGinGet[Req any](name string, service func(ctx context.Context, req Req) (any, error)) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		var req Req
		err := c.ShouldBindQuery(&req)
		if err != nil {
			logrus.WithContext(ctx).WithFields(logrus.Fields{"err": err}).Error(fmt.Sprintf("%s，请求参数解析异常", name))
			c.JSON(http.StatusOK, NewHttpRespByErr(nil, err))
			return
		}
		logrus.WithContext(ctx).WithFields(logrus.Fields{"req": req}).Info(name)
		c.JSON(http.StatusOK, NewHttpRespByErr(service(ctx, req)))
	}
}
func NewGinPost[Req any](name string, service func(ctx context.Context, req Req) (any, error)) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		var req Req
		err := c.ShouldBindJSON(&req)
		if err != nil {
			logrus.WithContext(ctx).WithFields(logrus.Fields{"err": err}).Error(fmt.Sprintf("%s，请求参数解析异常", name))
			c.JSON(http.StatusOK, NewHttpRespByErr(nil, err))
			return
		}
		logrus.WithContext(ctx).WithFields(logrus.Fields{"req": req}).Info(name)
		c.JSON(http.StatusOK, NewHttpRespByErr(service(ctx, req)))
	}
}
