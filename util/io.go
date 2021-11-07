package util

import (
	"bytes"
	"context"
	"crypto/md5"
	"encoding/csv"
	"encoding/hex"
	"fmt"
	"github.com/gocarina/gocsv"
	"github.com/sirupsen/logrus"
	"io"
	"io/ioutil"
	"os"
	"path"
	"strings"
)

func ExistPath(ctx context.Context, path string) (bool, os.FileInfo) {
	fileInfo, err := os.Stat(path)
	return err == nil || os.IsExist(err), fileInfo
}

func ExistAndIsFolder(ctx context.Context, folderPath string) (bool, os.FileInfo) {
	exist, fileInfo := ExistPath(ctx, folderPath)
	return exist && fileInfo.IsDir(), fileInfo
}

func ExistAndIsFile(ctx context.Context, filePath string) (bool, os.FileInfo) {
	exist, fileInfo := ExistPath(ctx, filePath)
	return exist && !fileInfo.IsDir(), fileInfo
}

func openFile(ctx context.Context, filePath string) (*os.File, error) {
	file, err := os.OpenFile(filePath, os.O_RDWR|os.O_TRUNC, 0666)
	if err != nil {
		logrus.WithContext(ctx).WithFields(logrus.Fields{"filePath": filePath, "err": err}).Error("文件打开异常")
		return nil, fmt.Errorf("文件打开异常: %+v", err)
	}
	return file, nil
}

func createFile(ctx context.Context, filePath string) (*os.File, error) {
	folderPath, _ := path.Split(filePath)
	if folderPath != "" {
		logrus.WithContext(ctx).WithFields(logrus.Fields{"folderPath": folderPath}).Info("文件父文件夹")
		err := CreateFolderPath(ctx, folderPath)
		if err != nil {
			return nil, err
		}
	}
	file, err := os.Create(filePath)
	if err != nil {
		logrus.WithContext(ctx).WithFields(logrus.Fields{"filePath": filePath, "err": err}).Error("文件创建异常")
		return nil, fmt.Errorf("文件创建异常: %+v", err)
	}
	return file, nil
}

func CreateFolderPath(ctx context.Context, folderPath string) error {
	err := os.MkdirAll(folderPath, 0777)
	if err != nil {
		logrus.WithContext(ctx).WithFields(logrus.Fields{"folderPath": folderPath, "err": err}).Error("创建文件夹异常")
		return fmt.Errorf("创建文件夹异常: %+v", err)
	}
	return nil
}

func GetFile(ctx context.Context, filePath string) (*os.File, error) {
	exist, fileInfo := ExistPath(ctx, filePath)
	if !exist {
		return createFile(ctx, filePath)
	}
	if fileInfo != nil && fileInfo.IsDir() {
		logrus.WithContext(ctx).WithFields(logrus.Fields{"filePath": filePath}).Error("该路径为文件夹，获取文件失败")
		return nil, fmt.Errorf("该路径为文件夹，获取文件失败")
	}
	return openFile(ctx, filePath)
}

func WriteFileWithData(ctx context.Context, filePath string, bytes []byte) error {
	file, err := GetFile(ctx, filePath)
	if err != nil {
		return err
	}
	defer func(filePath string, file *os.File) {
		err := file.Close()
		if err != nil {
			logrus.WithContext(ctx).WithFields(logrus.Fields{"filePath": filePath, "err": err}).Error("文件关闭异常")
		}
	}(filePath, file)
	written, err := file.Write(bytes)
	if err != nil {
		logrus.WithContext(ctx).WithFields(logrus.Fields{"filePath": filePath, "err": err}).Error("文件写入异常")
		return fmt.Errorf("文件写入异常: %+v", err)
	} else {
		logrus.WithContext(ctx).WithFields(logrus.Fields{"filePath": filePath, "written": written}).Info("文件写入完成")
	}
	return nil
}

func WriteFileWithString(ctx context.Context, filePath string, text string) error {
	return WriteFileWithData(ctx, filePath, []byte(text))
}

func WriteFileWithReader(ctx context.Context, filePath string, reader io.Reader) error {
	file, err := GetFile(ctx, filePath)
	if err != nil {
		return err
	}
	defer func(filePath string, file *os.File) {
		err := file.Close()
		if err != nil {
			logrus.WithContext(ctx).WithFields(logrus.Fields{"filePath": filePath, "err": err}).Error("文件关闭异常")
		}
	}(filePath, file)
	written, err := io.Copy(file, reader)
	if err != nil {
		logrus.WithContext(ctx).WithFields(logrus.Fields{"filePath": filePath, "err": err}).Error("文件写入异常")
		return fmt.Errorf("文件写入异常: %+v", err)
	} else {
		logrus.WithContext(ctx).WithFields(logrus.Fields{"filePath": filePath, "written": written}).Info("文件写入完成")
	}
	return nil
}

func ReadFileWithData(ctx context.Context, filePath string, defaultData []byte) ([]byte, error) {
	file, err := GetFile(ctx, filePath)
	if err != nil {
		return nil, err
	}
	defer func(filePath string, file *os.File) {
		err := file.Close()
		if err != nil {
			logrus.WithContext(ctx).WithFields(logrus.Fields{"filePath": filePath, "err": err}).Error("文件关闭异常")
		}
	}(filePath, file)
	data, err := ioutil.ReadAll(file)
	if err != nil {
		logrus.WithContext(ctx).WithFields(logrus.Fields{"filePath": filePath, "err": err}).Error("文件读取异常")
		return nil, fmt.Errorf("文件读取异常: %+v", err)
	} else {
		logrus.WithContext(ctx).WithFields(logrus.Fields{"filePath": filePath, "len(data)": len(data)}).Info("文件读取完成")
	}
	if len(data) == 0 {
		data = defaultData
	}
	return data, nil
}

func ReadFileWithString(ctx context.Context, filePath string, defaultText string) (string, error) {
	data, err := ReadFileWithData(ctx, filePath, []byte(defaultText))
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func ClearPath(ctx context.Context, fileOrFolderPath string) string {
	fileOrFolderPath = strings.ReplaceAll(fileOrFolderPath, "\\", "/")
	return path.Clean(fileOrFolderPath)
}

func GetFileMd5(ctx context.Context, filePath string) (string, error) {
	file, err := GetFile(ctx, filePath)
	if err != nil {
		return "", err
	}
	defer func(filePath string, file *os.File) {
		err := file.Close()
		if err != nil {
			logrus.WithContext(ctx).WithFields(logrus.Fields{"filePath": filePath, "err": err}).Error("文件关闭异常")
		}
	}(filePath, file)
	hash := md5.New()
	_, err = io.Copy(hash, file)
	if err != nil {
		logrus.WithContext(ctx).WithFields(logrus.Fields{"filePath": filePath, "err": err}).Error("文件计算MD5异常")
		return "", fmt.Errorf("文件计算MD5异常: %+v", err)
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}

func ReadCsvWithReader2Struct(ctx context.Context, reader io.Reader, list interface{}) error {
	err := gocsv.Unmarshal(reader, list)
	if err != nil {
		logrus.WithContext(ctx).WithFields(logrus.Fields{"err": err}).Error("解析CSV异常")
		return fmt.Errorf("解析CSV异常: %+v", err)
	}
	return nil
}

func ReadCsvWithData2Struct(ctx context.Context, data []byte, list interface{}) error {
	if len(data) == 0 {
		logrus.WithContext(ctx).WithFields(logrus.Fields{}).Warn("序列化CSV，为空")
		return nil
	}
	err := gocsv.UnmarshalBytes(data, list)
	if err != nil {
		logrus.WithContext(ctx).WithFields(logrus.Fields{"err": err}).Error("解析CSV异常")
		return fmt.Errorf("解析CSV异常: %+v", err)
	}
	return nil
}

func ReadCsvWithString2Struct(ctx context.Context, text string, list interface{}) error {
	text = strings.TrimSpace(text)
	if text == "" {
		logrus.WithContext(ctx).WithFields(logrus.Fields{}).Warn("序列化CSV，为空")
		return nil
	}
	return ReadCsvWithData2Struct(ctx, []byte(text), list)
}

func ReadCsvWithFile2Struct(ctx context.Context, filePath string, list interface{}) error {
	data, err := ReadFileWithData(ctx, filePath, []byte{})
	if err != nil {
		return err
	}
	return ReadCsvWithData2Struct(ctx, data, list)
}

func ReadCsvWithData2String(ctx context.Context, data []byte) ([][]string, error) {
	if len(data) == 0 {
		logrus.WithContext(ctx).WithFields(logrus.Fields{}).Warn("序列化CSV，为空")
		return nil, nil
	}
	reader := csv.NewReader(bytes.NewReader(data))
	list, err := reader.ReadAll()
	if err != nil {
		logrus.WithContext(ctx).WithFields(logrus.Fields{"err": err}).Error("解析CSV异常")
		return nil, fmt.Errorf("解析CSV异常: %+v", err)
	}
	return list, nil
}

func ReadCsvWithString2String(ctx context.Context, text string) ([][]string, error) {
	text = strings.TrimSpace(text)
	if text == "" {
		logrus.WithContext(ctx).WithFields(logrus.Fields{}).Warn("序列化CSV，为空")
		return nil, nil
	}
	return ReadCsvWithData2String(ctx, []byte(text))
}

func ReadCsvWithFile2String(ctx context.Context, filePath string) ([][]string, error) {
	data, err := ReadFileWithData(ctx, filePath, []byte{})
	if err != nil {
		return nil, err
	}
	return ReadCsvWithData2String(ctx, data)
}

func WriteCsv2WriterByStruct(ctx context.Context, list interface{}, writer io.Writer) error {
	err := gocsv.Marshal(list, writer)
	if err != nil {
		logrus.WithContext(ctx).WithFields(logrus.Fields{"err": err}).Error("序列化CSV异常")
		return fmt.Errorf("序列化CSV异常: %+v", err)
	}
	return nil
}

func WriteCsv2DataByStruct(ctx context.Context, list interface{}) ([]byte, error) {
	data, err := gocsv.MarshalBytes(list)
	if err != nil {
		logrus.WithContext(ctx).WithFields(logrus.Fields{"err": err}).Error("序列化CSV异常")
		return nil, fmt.Errorf("序列化CSV异常: %+v", err)
	}
	return data, nil
}

func WriteCsv2StringByStruct(ctx context.Context, list interface{}) (string, error) {
	text, err := gocsv.MarshalString(list)
	if err != nil {
		logrus.WithContext(ctx).WithFields(logrus.Fields{"err": err}).Error("序列化CSV异常")
		return "", fmt.Errorf("序列化CSV异常: %+v", err)
	}
	return text, nil
}

func WriteCsv2FileByStruct(ctx context.Context, list interface{}, filePath string) error {
	file, err := GetFile(ctx, filePath)
	if err != nil {
		return err
	}
	defer func(filePath string, file *os.File) {
		err := file.Close()
		if err != nil {
			logrus.WithContext(ctx).WithFields(logrus.Fields{"filePath": filePath, "err": err}).Error("文件关闭异常")
		}
	}(filePath, file)
	return WriteCsv2WriterByStruct(ctx, list, file)
}

func WriteCsv2DataByString(ctx context.Context, lines [][]string) ([]byte, error) {
	var buffer bytes.Buffer
	_, err := buffer.WriteString("\xEF\xBB\xBF")
	if err != nil {
		logrus.WithContext(ctx).WithFields(logrus.Fields{"err": err}).Error("序列化CSV异常")
		return nil, fmt.Errorf("序列化CSV异常: %+v", err)
	}
	writer := csv.NewWriter(&buffer)
	writer.Comma = ','
	writer.UseCRLF = true
	err = writer.WriteAll(lines)
	if err != nil {
		logrus.WithContext(ctx).WithFields(logrus.Fields{"err": err}).Error("序列化CSV异常")
		return nil, fmt.Errorf("序列化CSV异常: %+v", err)
	}
	writer.Flush()
	err = writer.Error()
	if err != nil {
		logrus.WithContext(ctx).WithFields(logrus.Fields{"err": err}).Error("序列化CSV异常")
		return nil, fmt.Errorf("序列化CSV异常: %+v", err)
	}
	return buffer.Bytes(), nil
}

func WriteCsv2DataByFile(ctx context.Context, lines [][]string, filePath string) error {
	data, err := WriteCsv2DataByString(ctx, lines)
	if err != nil {
		return err
	}
	return WriteFileWithData(ctx, filePath, data)
}

func RemoveFile(ctx context.Context, filePath string) error {
	exist, fileInfo := ExistPath(ctx, filePath)
	if !exist {
		logrus.WithFields(logrus.Fields{"filePath": filePath}).Warn("删除文件，文件不存在")
		return nil
	}
	if fileInfo != nil && fileInfo.IsDir() {
		files, err := ioutil.ReadDir(filePath)
		if err != nil {
			logrus.WithFields(logrus.Fields{"filePath": filePath, "err": err}).Error("删除文件，读取文件夹异常")
			return fmt.Errorf("删除文件，读取文件夹异常: %+v", err)
		}
		if len(files) > 0 {
			logrus.WithFields(logrus.Fields{"filePath": filePath}).Error("删除文件，文件夹不为空")
			return fmt.Errorf("删除文件，文件夹不为空")
		}
	}
	err := os.Remove(filePath)
	if err != nil {
		logrus.WithContext(ctx).WithFields(logrus.Fields{"filePath": filePath}).Error("删除文件，异常")
		return fmt.Errorf("删除文件，异常: %+v", err)
	}
	return nil
}
