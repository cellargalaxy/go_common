package util

import (
	"bytes"
	"context"
	"encoding/csv"
	"io"
	"strings"

	"github.com/gocarina/gocsv"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

func CsvReader2Strs(ctx context.Context, reader io.Reader) ([][]string, error) {
	read := csv.NewReader(reader)
	read.FieldsPerRecord = -1
	list, err := read.ReadAll()
	if err != nil {
		logrus.WithContext(ctx).WithFields(logrus.Fields{"err": err}).Error("解析CSV异常")
		return nil, errors.Errorf("解析CSV异常: %+v", err)
	}
	return list, nil
}
func CsvReader2Struct(ctx context.Context, reader io.Reader, list interface{}) (err error) {
	defer Defer(func(panic any, stack string) {
		if panic != nil {
			logrus.WithContext(ctx).WithFields(logrus.Fields{"panic": panic, "stack": stack}).Error("解析CSV异常")
			err = errors.Errorf("解析CSV异常: %+v", panic)
		}
	})

	err = gocsv.Unmarshal(reader, list)
	if err != nil {
		logrus.WithContext(ctx).WithFields(logrus.Fields{"err": err}).Error("解析CSV异常")
		return errors.Errorf("解析CSV异常: %+v", err)
	}
	return nil
}

func CsvData2Strs(ctx context.Context, data []byte) ([][]string, error) {
	if len(data) == 0 {
		logrus.WithContext(ctx).WithFields(logrus.Fields{}).Warn("序列化CSV，为空")
		return nil, nil
	}
	return CsvReader2Strs(ctx, bytes.NewReader(data))
}
func CsvData2Struct(ctx context.Context, data []byte, list interface{}) (err error) {
	defer Defer(func(panic any, stack string) {
		if panic != nil {
			logrus.WithContext(ctx).WithFields(logrus.Fields{"panic": panic, "stack": stack}).Error("解析CSV异常")
			err = errors.Errorf("解析CSV异常: %+v", panic)
		}
	})

	if len(data) == 0 {
		logrus.WithContext(ctx).WithFields(logrus.Fields{}).Warn("序列化CSV，为空")
		return nil
	}
	err = gocsv.UnmarshalBytes(data, list)
	if err != nil {
		logrus.WithContext(ctx).WithFields(logrus.Fields{"err": err}).Error("解析CSV异常")
		return errors.Errorf("解析CSV异常: %+v", err)
	}
	return nil
}

func CsvStr2Strs(ctx context.Context, text string) ([][]string, error) {
	text = strings.TrimSpace(text)
	if text == "" {
		logrus.WithContext(ctx).WithFields(logrus.Fields{}).Warn("序列化CSV，为空")
		return nil, nil
	}
	return CsvData2Strs(ctx, []byte(text))
}
func CsvStr2Struct(ctx context.Context, text string, list interface{}) error {
	text = strings.TrimSpace(text)
	if text == "" {
		logrus.WithContext(ctx).WithFields(logrus.Fields{}).Warn("序列化CSV，为空")
		return nil
	}
	return CsvData2Struct(ctx, []byte(text), list)
}

func CsvFile2Strs(ctx context.Context, filePath string) ([][]string, error) {
	data, err := ReadFile2Data(ctx, filePath, nil)
	if err != nil {
		return nil, err
	}
	return CsvData2Strs(ctx, data)
}
func CsvFile2Struct(ctx context.Context, filePath string, list interface{}) error {
	data, err := ReadFile2Data(ctx, filePath, nil)
	if err != nil {
		return err
	}
	return CsvData2Struct(ctx, data, list)
}

func CsvStrs2Data(ctx context.Context, lines [][]string) ([]byte, error) {
	var buffer bytes.Buffer
	_, err := buffer.WriteString("") //"\xEF\xBB\xBF"
	if err != nil {
		logrus.WithContext(ctx).WithFields(logrus.Fields{"err": err}).Error("序列化CSV异常")
		return nil, errors.Errorf("序列化CSV异常: %+v", err)
	}
	writer := csv.NewWriter(&buffer)
	writer.Comma = ','
	writer.UseCRLF = true
	err = writer.WriteAll(lines)
	if err != nil {
		logrus.WithContext(ctx).WithFields(logrus.Fields{"err": err}).Error("序列化CSV异常")
		return nil, errors.Errorf("序列化CSV异常: %+v", err)
	}
	writer.Flush()
	err = writer.Error()
	if err != nil {
		logrus.WithContext(ctx).WithFields(logrus.Fields{"err": err}).Error("序列化CSV异常")
		return nil, errors.Errorf("序列化CSV异常: %+v", err)
	}
	return buffer.Bytes(), nil
}
func CsvStrs2Str(ctx context.Context, lines [][]string) (string, error) {
	data, err := CsvStrs2Data(ctx, lines)
	if err != nil {
		return "", err
	}
	return string(data), err
}
func CsvStrs2File(ctx context.Context, lines [][]string, filePath string) error {
	data, err := CsvStrs2Data(ctx, lines)
	if err != nil {
		return err
	}
	return WriteData2File(ctx, data, filePath)
}
func CsvStrs2Writer(ctx context.Context, lines [][]string, writer io.Writer) error {
	data, err := CsvStrs2Data(ctx, lines)
	if err != nil {
		return err
	}
	_, err = writer.Write(data)
	if err != nil {
		return err
	}
	return nil
}
func CsvStrs2Struct(ctx context.Context, lines [][]string, list interface{}) error {
	data, err := CsvStrs2Data(ctx, lines)
	if err != nil {
		return err
	}
	err = CsvData2Struct(ctx, data, list)
	if err != nil {
		return err
	}
	return nil
}

func CsvStruct2Data(ctx context.Context, list interface{}) (data []byte, err error) {
	defer Defer(func(panic any, stack string) {
		if panic != nil {
			logrus.WithContext(ctx).WithFields(logrus.Fields{"panic": panic, "stack": stack}).Error("序列化CSV异常")
			err = errors.Errorf("序列化CSV异常: %+v", panic)
		}
	})

	data, err = gocsv.MarshalBytes(list)
	if err != nil {
		logrus.WithContext(ctx).WithFields(logrus.Fields{"err": err}).Error("序列化CSV异常")
		return nil, errors.Errorf("序列化CSV异常: %+v", err)
	}
	return data, nil
}
func CsvStruct2Str(ctx context.Context, list interface{}) (text string, err error) {
	defer Defer(func(panic any, stack string) {
		if panic != nil {
			logrus.WithContext(ctx).WithFields(logrus.Fields{"panic": panic, "stack": stack}).Error("序列化CSV异常")
			err = errors.Errorf("序列化CSV异常: %+v", panic)
		}
	})

	text, err = gocsv.MarshalString(list)
	if err != nil {
		logrus.WithContext(ctx).WithFields(logrus.Fields{"err": err}).Error("序列化CSV异常")
		return "", errors.Errorf("序列化CSV异常: %+v", err)
	}
	return text, nil
}
func CsvStruct2File(ctx context.Context, list interface{}, filePath string) error {
	data, err := CsvStruct2Data(ctx, list)
	if err != nil {
		return err
	}
	return WriteData2File(ctx, data, filePath)
}
func CsvStruct2Writer(ctx context.Context, list interface{}, writer io.Writer) (err error) {
	defer Defer(func(panic any, stack string) {
		if panic != nil {
			logrus.WithContext(ctx).WithFields(logrus.Fields{"panic": panic, "stack": stack}).Error("序列化CSV异常")
			err = errors.Errorf("序列化CSV异常: %+v", panic)
		}
	})

	err = gocsv.Marshal(list, writer)
	if err != nil {
		logrus.WithContext(ctx).WithFields(logrus.Fields{"err": err}).Error("序列化CSV异常")
		return errors.Errorf("序列化CSV异常: %+v", err)
	}
	return nil
}
func CsvStruct2Strs(ctx context.Context, list interface{}) ([][]string, error) {
	data, err := CsvStruct2Data(ctx, list)
	if err != nil {
		return nil, err
	}
	return CsvData2Strs(ctx, data)
}
