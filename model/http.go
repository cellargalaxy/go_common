package model

import (
	"context"
	"fmt"
	"github.com/golang-jwt/jwt"
	json "github.com/json-iterator/go"
)

type Claims struct {
	jwt.StandardClaims
	Ip         string `json:"ip,omitempty"`
	ServerName string `json:"sn,omitempty"`
	LogId      int64  `json:"logid,omitempty"`
	ReqId      string `json:"reqid,omitempty"`
	Uri        string `json:"uri,omitempty"`
}

func (this Claims) String() string {
	data, _ := json.MarshalToString(this)
	return data
}

type HttpResponse struct {
	Code int         `json:"code"`
	Msg  string      `json:"msg"`
	Data interface{} `json:"data"`
}

func (this HttpResponse) String() string {
	data, _ := json.MarshalToString(this)
	return data
}
func (this *HttpResponse) Success(ctx context.Context) error {
	switch this.Code {
	case HttpSuccessCode, HttpReRequestCode:
		return nil
	default:
		return fmt.Errorf("HTTP响应失败: %+v", this)
	}
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
