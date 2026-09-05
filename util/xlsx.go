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
	//先判断异常再注册关闭，否则file为空指针时关闭会panic
	if err != nil {
		logrus.WithContext(ctx).WithFields(logrus.Fields{"err": err}).Error("创建xlsx异常")
		return nil, errors.Errorf("创建xlsx异常: %+v", err)
	}
	if file == nil {
		logrus.WithContext(ctx).WithFields(logrus.Fields{}).Error("创建xlsx为空")
		return nil, errors.Errorf("创建xlsx为空")
	}
	defer CloseIo(ctx, file)
	//读取端不能只认硬编码的 Sheet1：写入端(XlsxStrings2Data)固定创建 Sheet1，
	//但本函数的上游 XlsxFile2Strings 接收的是任意文件路径，
	//其它工具导出的xlsx首表常叫 Data、数据表 等，实测直接报 "sheet Sheet1 does not exist"，
	//即函数名承诺的"读xlsx"实际只能读本库自己写的文件。
	//这里保持"优先 Sheet1"以完全不改变既有可用行为，仅在 Sheet1 不存在时回退到首个工作表。
	sheetName := XlsxSheetNameDefault
	sheets := file.GetSheetList()
	if !Contain(ctx, sheets, XlsxSheetNameDefault) {
		if len(sheets) == 0 {
			logrus.WithContext(ctx).WithFields(logrus.Fields{}).Error("读取xlsx异常，无工作表")
			return nil, errors.Errorf("读取xlsx异常，无工作表")
		}
		sheetName = sheets[0]
		logrus.WithContext(ctx).WithFields(logrus.Fields{"sheetName": sheetName}).Info("读取xlsx，无Sheet1，取首个工作表")
	}
	rows, err := file.GetRows(sheetName)
	if err != nil {
		logrus.WithContext(ctx).WithFields(logrus.Fields{"err": err}).Error("读取xlsx异常")
		return nil, errors.Errorf("读取xlsx异常: %+v", err)
	}
	return rows, nil
}
