package util

import (
	"context"
	"github.com/go-resty/resty/v2"
	"github.com/pkg/errors"
	"testing"
)

type JsonMockResponse struct {
	Id int `json:"id"`
}

func (this *JsonMockResponse) HttpSuccess(ctx context.Context) error {
	if this.Id <= 0 {
		return errors.Errorf(`if this.Id <= 0 {`)
	}
	return nil
}

func TestHttpApiTry(t *testing.T) {
	ctx := GenCtx()

	var object JsonMockResponse
	err := HttpApiTry(ctx, "HttpApiTry", 0, SpiderSleepDefault, &object, func() (*resty.Response, error) {
		return GetHttpRequest(ctx).Get("https://jsonplaceholder.typicode.com/todos/1")
	})
	if err != nil {
		t.Errorf("%+v", err)
		return
	}
	if object.Id <= 0 {
		t.Errorf(`if object.Id <= 0 {`)
		return
	}
}
