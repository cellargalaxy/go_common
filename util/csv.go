package util

import (
	"bytes"
	"context"
	"encoding/csv"
	"github.com/gocarina/gocsv"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"io"
	"strings"
)

func CsvReader2Strings(ctx context.Context, reader io.Reader) ([][]string, error) {
	read := csv.NewReader(reader)
	list, err := read.ReadAll()
	if err != nil {
		logrus.WithContext(ctx).WithFields(logrus.Fields{"err": err}).Error("解析CSV异常")
		return nil, errors.Errorf("解析CSV异常: %+v", err)
	}
	return list, nil
}
func CsvReader2Struct(ctx context.Context, reader io.Reader, list interface{}) error {
	err := gocsv.Unmarshal(reader, list)
	if err != nil {
		logrus.WithContext(ctx).WithFields(logrus.Fields{"err": err}).Error("解析CSV异常")
		return errors.Errorf("解析CSV异常: %+v", err)
	}
	return nil
}

func CsvData2Strings(ctx context.Context, data []byte) ([][]string, error) {
	if len(data) == 0 {
		logrus.WithContext(ctx).WithFields(logrus.Fields{}).Warn("序列化CSV，为空")
		return nil, nil
	}
	return CsvReader2Strings(ctx, bytes.NewReader(data))
}
func CsvData2Struct(ctx context.Context, data []byte, list interface{}) error {
	if len(data) == 0 {
		logrus.WithContext(ctx).WithFields(logrus.Fields{}).Warn("序列化CSV，为空")
		return nil
	}
	err := gocsv.UnmarshalBytes(data, list)
	if err != nil {
		logrus.WithContext(ctx).WithFields(logrus.Fields{"err": err}).Error("解析CSV异常")
		return errors.Errorf("解析CSV异常: %+v", err)
	}
	return nil
}

func CsvString2Strings(ctx context.Context, text string) ([][]string, error) {
	text = strings.TrimSpace(text)
	if text == "" {
		logrus.WithContext(ctx).WithFields(logrus.Fields{}).Warn("序列化CSV，为空")
		return nil, nil
	}
	return CsvData2Strings(ctx, []byte(text))
}
func CsvString2Struct(ctx context.Context, text string, list interface{}) error {
	text = strings.TrimSpace(text)
	if text == "" {
		logrus.WithContext(ctx).WithFields(logrus.Fields{}).Warn("序列化CSV，为空")
		return nil
	}
	return CsvData2Struct(ctx, []byte(text), list)
}

func CsvFile2Strings(ctx context.Context, filePath string) ([][]string, error) {
	data, err := ReadFile2Data(ctx, filePath, nil)
	if err != nil {
		return nil, err
	}
	return CsvData2Strings(ctx, data)
}
func CsvFile2Struct(ctx context.Context, filePath string, list interface{}) error {
	data, err := ReadFile2Data(ctx, filePath, nil)
	if err != nil {
		return err
	}
	return CsvData2Struct(ctx, data, list)
}

func CsvStrings2Data(ctx context.Context, lines [][]string) ([]byte, error) {
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
func CsvStrings2String(ctx context.Context, lines [][]string) (string, error) {
	data, err := CsvStrings2Data(ctx, lines)
	if err != nil {
		return "", err
	}
	return string(data), err
}
func CsvStrings2File(ctx context.Context, lines [][]string, filePath string) error {
	data, err := CsvStrings2Data(ctx, lines)
	if err != nil {
		return err
	}
	return WriteData2File(ctx, data, filePath)
}
func CsvStrings2Writer(ctx context.Context, lines [][]string, writer io.Writer) error {
	data, err := CsvStrings2Data(ctx, lines)
	if err != nil {
		return err
	}
	_, err = writer.Write(data)
	if err != nil {
		return err
	}
	return nil
}
func CsvStrings2Struct(ctx context.Context, lines [][]string, list interface{}) error {
	data, err := CsvStrings2Data(ctx, lines)
	if err != nil {
		return err
	}
	err = CsvData2Struct(ctx, data, list)
	if err != nil {
		return err
	}
	return nil
}

func CsvStruct2Data(ctx context.Context, list interface{}) ([]byte, error) {
	data, err := gocsv.MarshalBytes(list)
	if err != nil {
		logrus.WithContext(ctx).WithFields(logrus.Fields{"err": err}).Error("序列化CSV异常")
		return nil, errors.Errorf("序列化CSV异常: %+v", err)
	}
	return data, nil
}
func CsvStruct2String(ctx context.Context, list interface{}) (string, error) {
	text, err := gocsv.MarshalString(list)
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
func CsvStruct2Writer(ctx context.Context, list interface{}, writer io.Writer) error {
	err := gocsv.Marshal(list, writer)
	if err != nil {
		logrus.WithContext(ctx).WithFields(logrus.Fields{"err": err}).Error("序列化CSV异常")
		return errors.Errorf("序列化CSV异常: %+v", err)
	}
	return nil
}
func CsvStruct2Strings(ctx context.Context, list interface{}) ([][]string, error) {
	data, err := CsvStruct2Data(ctx, list)
	if err != nil {
		return nil, err
	}
	return CsvData2Strings(ctx, data)
}
