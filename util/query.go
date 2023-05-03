package util

import (
	"context"
	"github.com/hetiansu5/urlquery"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

func QueryStruct2Data(ctx context.Context, x interface{}) []byte {
	bytes, err := urlquery.Marshal(x)
	if err != nil {
		logrus.WithContext(ctx).WithFields(logrus.Fields{"x": x, "err": errors.WithStack(err)}).Error("序列化query异常")
	}
	return bytes
}
func QueryStruct2String(ctx context.Context, x interface{}) string {
	bytes := QueryStruct2Data(ctx, x)
	return string(bytes)
}

func QueryData2Struct(ctx context.Context, data []byte, v interface{}) error {
	err := urlquery.Unmarshal(data, v)
	if err != nil {
		logrus.WithContext(ctx).WithFields(logrus.Fields{"data": string(data), "err": errors.WithStack(err)}).Error("反序列化query异常")
		return errors.Errorf("反序列化query异常: %+v", err)
	}
	return nil
}
func QueryString2Struct(ctx context.Context, data string, v interface{}) error {
	return QueryData2Struct(ctx, []byte(data), v)
}
