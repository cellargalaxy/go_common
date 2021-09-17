package model

import (
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt"
)

type Claims struct {
	jwt.StandardClaims
	AllowReRequest bool   `json:"allow_re_request,omitempty"`
	RequestId      string `json:"request_id,omitempty"`
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
