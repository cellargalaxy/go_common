package util

import (
	"context"

	"github.com/hetiansu5/urlquery"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

func QueryStruct2Data(ctx context.Context, x interface{}) (data []byte) {
	defer Defer(func(panic any, stack string) {
		if panic != nil {
			logrus.WithContext(ctx).WithFields(logrus.Fields{"panic": panic, "stack": stack}).Error("序列化query异常")
		}
	})

	data, err := urlquery.Marshal(x)
	if err != nil {
		logrus.WithContext(ctx).WithFields(logrus.Fields{"x": x, "err": errors.WithStack(err)}).Error("序列化query异常")
		return nil
	}
	return data
}
func QueryStruct2Str(ctx context.Context, x interface{}) string {
	data := QueryStruct2Data(ctx, x)
	return string(data)
}

func QueryData2Struct(ctx context.Context, data []byte, v interface{}) (err error) {
	defer Defer(func(panic any, stack string) {
		if panic != nil {
			logrus.WithContext(ctx).WithFields(logrus.Fields{"panic": panic, "stack": stack}).Error("反序列化query异常")
			err = errors.Errorf("反序列化query异常: %+v", panic)
		}
	})

	err = urlquery.Unmarshal(data, v)
	if err != nil {
		logrus.WithContext(ctx).WithFields(logrus.Fields{"data": string(data), "err": errors.WithStack(err)}).Error("反序列化query异常")
		return errors.Errorf("反序列化query异常: %+v", err)
	}
	return nil
}
func QueryStr2Struct(ctx context.Context, data string, v interface{}) error {
	return QueryData2Struct(ctx, []byte(data), v)
}
