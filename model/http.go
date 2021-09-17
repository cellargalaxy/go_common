package model

import (
	"github.com/cellargalaxy/go_common/util"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt"
)

type Claims struct {
	jwt.StandardClaims
	AllowReRequest bool   `json:"allow_re_request,omitempty"`
	RequestId      string `json:"request_id,omitempty"`
}

func (this Claims) String() string {
	return util.ToJsonString(this)
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

func (this HttpRequestParam) String() string {
	return util.ToJsonString(this)
}
