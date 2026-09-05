package util

import (
	"context"
	"github.com/hetiansu5/urlquery"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

func QueryStruct2Data(ctx context.Context, x interface{}) (data []byte) {
	//urlquery 对nil等非法入参是直接panic（reflect.Value.Type on zero Value）而非返回error，
	//这与 JsonStruct2Data、YamlStruct2Data 的"记日志并返回空"约定不一致，
	//会让调用方在序列化日志字段这类旁路逻辑中被动崩溃，故在此兜底。
	defer func() {
		if r := recover(); r != nil {
			logrus.WithContext(ctx).WithFields(logrus.Fields{"x": x, "err": errors.Errorf("%+v", r)}).Error("序列化query异常")
			data = nil
		}
	}()
	data, err := urlquery.Marshal(x)
	if err != nil {
		logrus.WithContext(ctx).WithFields(logrus.Fields{"x": x, "err": errors.WithStack(err)}).Error("序列化query异常")
		return nil
	}
	return data
}
func QueryStruct2String(ctx context.Context, x interface{}) string {
	data := QueryStruct2Data(ctx, x)
	return string(data)
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
