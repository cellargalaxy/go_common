package util

import (
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
)

func YamlStruct2Data(x interface{}) []byte {
	bytes, err := yaml.Marshal(x)
	if err != nil {
		logrus.WithFields(logrus.Fields{"x": x, "err": errors.WithStack(err)}).Error("序列化yaml异常")
	}
	return bytes
}

func YamlStruct2String(x interface{}) string {
	bytes := YamlStruct2Data(x)
	return string(bytes)
}

func YamlData2Struct(data []byte, v interface{}) error {
	err := yaml.Unmarshal(data, v)
	if err != nil {
		logrus.WithFields(logrus.Fields{"data": string(data), "err": errors.WithStack(err)}).Error("反序列化yaml异常")
		return errors.Errorf("反序列化yaml异常: %+v", err)
	}
	return nil
}

func YamlString2Struct(data string, v interface{}) error {
	return YamlData2Struct([]byte(data), v)
}
