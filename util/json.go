package util

import (
	json "github.com/json-iterator/go"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

func ToJson(x interface{}) []byte {
	bytes, err := json.Marshal(x)
	if err != nil {
		logrus.WithFields(logrus.Fields{"x": x, "err": errors.WithStack(err)}).Error("序列化json异常")
	}
	return bytes
}

func ToJsonIndent(x interface{}) []byte {
	bytes, err := json.MarshalIndent(x, "", "  ")
	if err != nil {
		logrus.WithFields(logrus.Fields{"x": x, "err": errors.WithStack(err)}).Error("序列化json异常")
	}
	return bytes
}

func ToJsonString(x interface{}) string {
	bytes := ToJson(x)
	return string(bytes)
}

func ToJsonIndentString(x interface{}) string {
	bytes := ToJsonIndent(x)
	return string(bytes)
}

func UnmarshalJson(data []byte, v interface{}) error {
	err := json.Unmarshal(data, v)
	if err != nil {
		logrus.WithFields(logrus.Fields{"data": string(data), "err": errors.WithStack(err)}).Error("反序列化json异常")
		return errors.Errorf("反序列化json异常: %+v", err)
	}
	return nil
}

func UnmarshalJsonString(data string, v interface{}) error {
	return UnmarshalJson([]byte(data), v)
}
