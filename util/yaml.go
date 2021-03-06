package util

import (
	"fmt"
	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
)

func ToYaml(x interface{}) []byte {
	bytes, err := yaml.Marshal(x)
	if err != nil {
		logrus.WithFields(logrus.Fields{"err": err}).Error("序列化yaml异常")
	}
	return bytes
}

func ToYamlString(x interface{}) string {
	bytes := ToYaml(x)
	return string(bytes)
}

func UnmarshalYaml(data []byte, v interface{}) error {
	err := yaml.Unmarshal(data, v)
	if err != nil {
		logrus.WithFields(logrus.Fields{"err": err}).Error("反序列化yaml异常")
		return fmt.Errorf("反序列化yaml异常: %+v", err)
	}
	return nil
}

func UnmarshalYamlString(data string, v interface{}) error {
	return UnmarshalYaml([]byte(data), v)
}
