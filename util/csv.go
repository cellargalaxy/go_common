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
	//FieldsPerRecord<0 表示不校验各行列数是否一致。
	//本库写入端 CsvStrings2Data 用的是 writer.WriteAll，对参差不齐的行照写不误
	//（Table 明确支持参差不齐的行，见 Table.AddCol 对短行/长行的处理），
	//而 csv.Reader 默认以首行列数为准，遇到列数不同的行直接报
	//"wrong number of fields"，导致本库自己写出的CSV自己读不回来：
	//实测 [[a b],[c d e]] 经 CsvStrings2Data -> CsvData2Strings 报错且数据全丢。
	//读取端放宽为不校验列数，与写入端的容忍度对齐。
	read.FieldsPerRecord = -1
	list, err := read.ReadAll()
	if err != nil {
		logrus.WithContext(ctx).WithFields(logrus.Fields{"err": err}).Error("解析CSV异常")
		return nil, errors.Errorf("解析CSV异常: %+v", err)
	}
	return list, nil
}
func CsvReader2Struct(ctx context.Context, reader io.Reader, list interface{}) (err error) {
	//gocsv 在目标为nil或不可寻址时会panic（reflect.Value.Type on zero Value），
	//与本文件其余函数"返回error"的约定不一致，故在此兜底转为error
	defer func() {
		if r := recover(); r != nil {
			logrus.WithContext(ctx).WithFields(logrus.Fields{"err": errors.Errorf("%+v", r)}).Error("解析CSV异常")
			err = errors.Errorf("解析CSV异常: %+v", r)
		}
	}()
	err = gocsv.Unmarshal(reader, list)
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
func CsvData2Struct(ctx context.Context, data []byte, list interface{}) (err error) {
	if len(data) == 0 {
		logrus.WithContext(ctx).WithFields(logrus.Fields{}).Warn("序列化CSV，为空")
		return nil
	}
	//与 CsvReader2Struct 同因兜底：gocsv 对非法目标会panic
	defer func() {
		if r := recover(); r != nil {
			logrus.WithContext(ctx).WithFields(logrus.Fields{"err": errors.Errorf("%+v", r)}).Error("解析CSV异常")
			err = errors.Errorf("解析CSV异常: %+v", r)
		}
	}()
	err = gocsv.UnmarshalBytes(data, list)
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

func CsvStruct2Data(ctx context.Context, list interface{}) (data []byte, err error) {
	//与 CsvReader2Struct 同因兜底：gocsv 对nil等非法入参会panic
	//（reflect.Value.Type on zero Value），须与本文件"返回error"的约定保持一致
	defer func() {
		if r := recover(); r != nil {
			logrus.WithContext(ctx).WithFields(logrus.Fields{"err": errors.Errorf("%+v", r)}).Error("序列化CSV异常")
			data = nil
			err = errors.Errorf("序列化CSV异常: %+v", r)
		}
	}()
	data, err = gocsv.MarshalBytes(list)
	if err != nil {
		logrus.WithContext(ctx).WithFields(logrus.Fields{"err": err}).Error("序列化CSV异常")
		return nil, errors.Errorf("序列化CSV异常: %+v", err)
	}
	return data, nil
}
func CsvStruct2String(ctx context.Context, list interface{}) (text string, err error) {
	//同上，兜底gocsv对非法入参的panic
	defer func() {
		if r := recover(); r != nil {
			logrus.WithContext(ctx).WithFields(logrus.Fields{"err": errors.Errorf("%+v", r)}).Error("序列化CSV异常")
			text = ""
			err = errors.Errorf("序列化CSV异常: %+v", r)
		}
	}()
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
	//同上，兜底gocsv对非法入参的panic
	defer func() {
		if r := recover(); r != nil {
			logrus.WithContext(ctx).WithFields(logrus.Fields{"err": errors.Errorf("%+v", r)}).Error("序列化CSV异常")
			err = errors.Errorf("序列化CSV异常: %+v", r)
		}
	}()
	err = gocsv.Marshal(list, writer)
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
