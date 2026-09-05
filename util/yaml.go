package util

import (
	"context"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
)

func YamlStruct2Data(ctx context.Context, x interface{}) (data []byte) {
	//yaml.v2 对不支持的类型（func、chan等）是直接panic而非返回error，
	//这与 JsonStruct2Data、QueryStruct2Data 的"记日志并返回空"约定不一致，
	//会让调用方在序列化日志字段这类旁路逻辑中被动崩溃，故在此兜底。
	defer func() {
		if r := recover(); r != nil {
			logrus.WithContext(ctx).WithFields(logrus.Fields{"x": x, "err": errors.Errorf("%+v", r)}).Error("序列化yaml异常")
			data = nil
		}
	}()
	data, err := yaml.Marshal(x)
	if err != nil {
		logrus.WithContext(ctx).WithFields(logrus.Fields{"x": x, "err": errors.WithStack(err)}).Error("序列化yaml异常")
		return nil
	}
	return data
}

func YamlStruct2String(ctx context.Context, x interface{}) string {
	bytes := YamlStruct2Data(ctx, x)
	return string(bytes)
}

func YamlData2Struct(ctx context.Context, data []byte, v interface{}) (err error) {
	//yaml.v2 在目标非指针或为nil时会panic（reflect.Value.Set using unaddressable value），
	//而 JsonData2Struct、QueryData2Struct 均返回error，故在此兜底转为error保持一致。
	defer func() {
		if r := recover(); r != nil {
			logrus.WithContext(ctx).WithFields(logrus.Fields{"data": string(data), "err": errors.Errorf("%+v", r)}).Error("反序列化yaml异常")
			err = errors.Errorf("反序列化yaml异常: %+v", r)
		}
	}()
	err = yaml.Unmarshal(data, v)
	if err != nil {
		logrus.WithContext(ctx).WithFields(logrus.Fields{"data": string(data), "err": errors.WithStack(err)}).Error("反序列化yaml异常")
		return errors.Errorf("反序列化yaml异常: %+v", err)
	}
	return nil
}

func YamlString2Struct(ctx context.Context, data string, v interface{}) error {
	return YamlData2Struct(ctx, []byte(data), v)
}
