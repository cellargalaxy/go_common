package model

import (
	"github.com/golang-jwt/jwt"
	json "github.com/json-iterator/go"
)

type Claims struct {
	jwt.StandardClaims
	Ip         string `json:"ip,omitempty"`    //某台服务器
	ServerName string `json:"sn,omitempty"`    //的某个服务
	LogId      int64  `json:"logid,omitempty"` //在某次调用链中
	Uri        string `json:"uri,omitempty"`   //向某个接口
	ReqId      int64  `json:"reqid,omitempty"` //发起的某次请求
}

func (this Claims) String() string {
	data, _ := json.MarshalToString(this)
	return data
}

type HttpData struct {
	Object any   `json:"object"`
	Count  int64 `json:"count"`
}

type HttpResp struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
	Data any    `json:"data"`
}

func (this HttpResp) String() string {
	data, _ := json.MarshalToString(this)
	return data
}

type PingReq struct {
}

type PingData struct {
	Ip         string `json:"ip"` //某台服务器
	ServerName string `json:"sn"` //的某个服务
	Timestamp  int64  `json:"ts"` //的响应
}
