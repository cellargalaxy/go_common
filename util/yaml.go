package util

import (
	"context"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
)

func YamlStruct2Data(ctx context.Context, x interface{}) (data []byte) {
	defer Defer(func(err interface{}, stack string) {
		if err != nil {
			logrus.WithContext(ctx).WithFields(logrus.Fields{"err": err, "stack": stack}).Error("序列化yaml异常")
			err = errors.Errorf("序列化yaml异常: %+v", err)
		}
	})

	data, err := yaml.Marshal(x)
	if err != nil {
		logrus.WithContext(ctx).WithFields(logrus.Fields{"x": x, "err": errors.WithStack(err)}).Error("序列化yaml异常")
		return nil
	}
	return data
}

func YamlStruct2Str(ctx context.Context, x interface{}) string {
	bytes := YamlStruct2Data(ctx, x)
	return string(bytes)
}

func YamlData2Struct(ctx context.Context, data []byte, v interface{}) (err error) {
	defer Defer(func(err interface{}, stack string) {
		if err != nil {
			logrus.WithContext(ctx).WithFields(logrus.Fields{"err": err, "stack": stack}).Error("反序列化yaml异常")
			err = errors.Errorf("反序列化yaml异常: %+v", err)
		}
	})

	err = yaml.Unmarshal(data, v)
	if err != nil {
		logrus.WithContext(ctx).WithFields(logrus.Fields{"data": string(data), "err": errors.WithStack(err)}).Error("反序列化yaml异常")
		return errors.Errorf("反序列化yaml异常: %+v", err)
	}
	return nil
}

func YamlStr2Struct(ctx context.Context, data string, v interface{}) error {
	return YamlData2Struct(ctx, []byte(data), v)
}
