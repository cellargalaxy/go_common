package util

import (
	json "github.com/json-iterator/go"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

func JsonStruct2Data(x interface{}) []byte {
	bytes, err := json.Marshal(x)
	if err != nil {
		logrus.WithFields(logrus.Fields{"x": x, "err": errors.WithStack(err)}).Error("序列化json异常")
	}
	return bytes
}
func JsonStruct2DataIndent(x interface{}) []byte {
	bytes, err := json.MarshalIndent(x, "", "  ")
	if err != nil {
		logrus.WithFields(logrus.Fields{"x": x, "err": errors.WithStack(err)}).Error("序列化json异常")
	}
	return bytes
}

func JsonStruct2String(x interface{}) string {
	bytes := JsonStruct2Data(x)
	return string(bytes)
}
func JsonStruct2StringIndent(x interface{}) string {
	bytes := JsonStruct2DataIndent(x)
	return string(bytes)
}

func JsonData2Struct(data []byte, v interface{}) error {
	err := json.Unmarshal(data, v)
	if err != nil {
		logrus.WithFields(logrus.Fields{"data": string(data), "err": errors.WithStack(err)}).Error("反序列化json异常")
		return errors.Errorf("反序列化json异常: %+v", err)
	}
	return nil
}
func JsonString2Struct(data string, v interface{}) error {
	return JsonData2Struct([]byte(data), v)
}
