package util

import (
	"context"
	"github.com/hetiansu5/urlquery"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

func ToQuery(ctx context.Context, x interface{}) []byte {
	bytes, err := urlquery.Marshal(x)
	if err != nil {
		logrus.WithFields(logrus.Fields{"x": x, "err": errors.WithStack(err)}).Error("序列化query异常")
	}
	return bytes
}

func ToQueryString(ctx context.Context, x interface{}) string {
	bytes := ToQuery(ctx, x)
	return string(bytes)
}

func UnmarshalQuery(ctx context.Context, data []byte, v interface{}) error {
	err := urlquery.Unmarshal(data, v)
	if err != nil {
		logrus.WithFields(logrus.Fields{"data": string(data), "err": errors.WithStack(err)}).Error("反序列化query异常")
		return errors.Errorf("反序列化query异常: %+v", err)
	}
	return nil
}

func UnmarshalQueryString(ctx context.Context, data string, v interface{}) error {
	return UnmarshalQuery(ctx, []byte(data), v)
}
