package model

import (
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
