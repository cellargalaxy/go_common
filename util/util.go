package util

import (
	"github.com/go-resty/resty/v2"
	"time"
)

var httpClient *resty.Client

func init() {
	httpClient = CreateNotTryHttpClient(time.Second * 5)

	initRegexp()
	initHttp()
	Init()
}

func Init() {

}
