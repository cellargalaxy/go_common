package model

import (
	"context"
	"github.com/golang-jwt/jwt"
	json "github.com/json-iterator/go"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
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
func (this *HttpResp) HttpSuccess(ctx context.Context) error {
	switch this.Code {
	case SuccessCode, ReRequestCode:
		return nil
	default:
		logrus.WithContext(ctx).WithFields(logrus.Fields{"this": this}).Error("HTTP响应失败")
		return errors.Errorf("HTTP响应失败: %+v", this)
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
