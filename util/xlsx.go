package util

import (
	"bytes"
	"context"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/xuri/excelize/v2"
)

const (
	XlsxSheetNameDefault = "Sheet1"
)

func XlsxStrings2File(ctx context.Context, lines [][]string, filePath string) error {
	data, err := XlsxStrings2Data(ctx, lines)
	if err != nil {
		return err
	}
	return WriteData2File(ctx, data, filePath)
}
func XlsxStrings2Data(ctx context.Context, lines [][]string) ([]byte, error) {
	file := excelize.NewFile()
	defer CloseIo(ctx, file)
	if file == nil {
		logrus.WithContext(ctx).WithFields(logrus.Fields{}).Error("创建xlsx为空")
		return nil, errors.Errorf("创建xlsx为空")
	}

	for i := range lines {
		for j := range lines[i] {
			cell, err := excelize.CoordinatesToCellName(j+1, i+1)
			if err != nil {
				logrus.WithContext(ctx).WithFields(logrus.Fields{"err": err}).Error("写入xlsx异常")
				return nil, errors.Errorf("写入xlsx异常: %+v", err)
			}
			err = file.SetCellStr(XlsxSheetNameDefault, cell, lines[i][j])
			if err != nil {
				logrus.WithContext(ctx).WithFields(logrus.Fields{"err": err}).Error("写入xlsx异常")
				return nil, errors.Errorf("写入xlsx异常: %+v", err)
			}
		}
	}

	buffer, err := file.WriteToBuffer()
	if err != nil {
		logrus.WithContext(ctx).WithFields(logrus.Fields{"err": err}).Error("写入xlsx异常")
		return nil, errors.Errorf("写入xlsx异常: %+v", err)
	}
	return buffer.Bytes(), nil
}

func XlsxFile2Strings(ctx context.Context, filePath string) ([][]string, error) {
	data, err := ReadFile2Data(ctx, filePath, nil)
	if err != nil {
		return nil, err
	}
	return XlsxData2Strings(ctx, data)
}
func XlsxData2Strings(ctx context.Context, data []byte) ([][]string, error) {
	buffer := bytes.NewBuffer(data)
	file, err := excelize.OpenReader(buffer)
	defer CloseIo(ctx, file)
	if err != nil {
		logrus.WithContext(ctx).WithFields(logrus.Fields{"err": err}).Error("创建xlsx异常")
		return nil, errors.Errorf("创建xlsx异常: %+v", err)
	}
	if file == nil {
		logrus.WithContext(ctx).WithFields(logrus.Fields{}).Error("创建xlsx为空")
		return nil, errors.Errorf("创建xlsx为空")
	}
	rows, err := file.GetRows(XlsxSheetNameDefault)
	if err != nil {
		logrus.WithContext(ctx).WithFields(logrus.Fields{"err": err}).Error("读取xlsx异常")
		return nil, errors.Errorf("读取xlsx异常: %+v", err)
	}
	return rows, nil
}
