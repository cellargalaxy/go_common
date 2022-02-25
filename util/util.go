package util

import (
	"crypto/tls"
	"github.com/go-resty/resty/v2"
	"time"
)

var httpClient *resty.Client

func init() {
	httpClient = resty.New().
		SetTimeout(5 * time.Second).
		SetTLSClientConfig(&tls.Config{InsecureSkipVerify: true})

	initRegexp()
	initHttp()
	Init()
}

func Init() {

}
