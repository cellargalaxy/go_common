package model

import (
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt"
)

type Claims struct {
	jwt.StandardClaims
	Caller         string `json:"caller,omitempty"`
	AllowReRequest bool   `json:"allow_re_request,omitempty"`
	RequestId      string `json:"request_id,omitempty"`
	CreateTime     int64  `json:"create_time,omitempty"`
}

type HttpValidateInter interface {
	GetSecret(c *gin.Context) string
	CreateClaims(c *gin.Context) *Claims
}

type HttpRequestParam struct {
	Url    string            `json:"url"`
	Header map[string]string `json:"header"`
	Body   string            `json:"body"`
}

type PingRequest struct {
}

type PingResponse struct {
	Ts         int64  `json:"ts"`
	ServerName string `json:"server_name"`
}
