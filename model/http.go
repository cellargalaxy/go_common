package model

import (
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt"
)

type Claims struct {
	jwt.StandardClaims
	Ip         string `json:"ip,omitempty"`
	ServerName string `json:"sn,omitempty"`
	LogId      int64  `json:"logid,omitempty"`
	ReqId      string `json:"reqid,omitempty"`
}

type HttpValidateInter interface {
	GetSecret(c *gin.Context) string
	CreateClaims(c *gin.Context) Claims
}

type HttpRequestParam struct {
	Url    string            `json:"url"`
	Header map[string]string `json:"header"`
	Body   string            `json:"body"`
}

type PingRequest struct {
}

type PingResponse struct {
	Timestamp  int64  `json:"ts"`
	ServerName string `json:"sn"`
}
