package util

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"github.com/pkg/errors"
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

func WriteFileWithBytesOrCreateIfNotExist(ctx context.Context, filePath string, bytes []byte) error {
	exist, _ := ExistPath(ctx, filePath)
	if !exist {
		return CreateFileWithBytes(ctx, filePath, bytes)
	}
	return writeFileWithBytes(ctx, filePath, bytes)
}

func WriteFileWithReaderOrCreateIfNotExist(ctx context.Context, filePath string, reader io.Reader) error {
	exist, _ := ExistPath(ctx, filePath)
	if !exist {
		return CreateFileWithReader(ctx, filePath, reader)
	}
	return writeFileWithReader(ctx, filePath, reader)
}

func ReadFileOrCreateIfNotExist(ctx context.Context, filePath string, defaultText string) (string, error) {
	exist, _ := ExistPath(ctx, filePath)
	if !exist {
		err := CreateFileWithBytes(ctx, filePath, []byte(defaultText))
		return defaultText, err
	}
	bytes, err := readFile(ctx, filePath)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

func CreateFileWithBytes(ctx context.Context, filePath string, bytes []byte) error {
	file, err := createFile(ctx, filePath)
	if err != nil {
		return err
	}
	defer file.Close()
	_, err = file.Write(bytes)
	if err != nil {
		logrus.WithContext(ctx).WithFields(logrus.Fields{"filePath": filePath, "err": err}).Error("写入文件初始数据异常")
		return errors.Errorf("写入文件初始数据异常: %+v", err)
	}
	return nil
}

func CreateFileWithReader(ctx context.Context, filePath string, reader io.Reader) error {
	file, err := createFile(ctx, filePath)
	if err != nil {
		return err
	}
	defer file.Close()
	_, err = io.Copy(file, reader)
	if err != nil {
		logrus.WithContext(ctx).WithFields(logrus.Fields{"filePath": filePath, "err": err}).Error("写入文件初始数据异常")
		return errors.Errorf("写入文件初始数据异常: %+v", err)
	}
	return nil
}

func writeFileWithBytes(ctx context.Context, filePath string, bytes []byte) error {
	err := ioutil.WriteFile(filePath, bytes, 0644)
	if err != nil {
		logrus.WithContext(ctx).WithFields(logrus.Fields{"filePath": filePath, "err": err}).Error("写入文件异常")
		return errors.Errorf("写入文件异常: %+v", err)
	}
	return nil
}

func writeFileWithReader(ctx context.Context, filePath string, reader io.Reader) error {
	file, err := os.Open(filePath)
	if err != nil {
		logrus.WithContext(ctx).WithFields(logrus.Fields{"filePath": filePath, "err": err}).Error("打开文件异常")
		return errors.Errorf("打开文件异常: %+v", err)
	}
	defer file.Close()
	_, err = io.Copy(file, reader)
	if err != nil {
		logrus.WithContext(ctx).WithFields(logrus.Fields{"filePath": filePath, "err": err}).Error("写入文件数据异常")
		return errors.Errorf("写入文件数据异常: %+v", err)
	}
	return nil
}

func createFile(ctx context.Context, filePath string) (*os.File, error) {
	folderPath, _ := path.Split(filePath)
	logrus.WithContext(ctx).WithFields(logrus.Fields{"folderPath": folderPath}).Info("文件父文件夹")
	if folderPath != "" {
		err := os.MkdirAll(folderPath, 0666)
		if err != nil {
			logrus.WithContext(ctx).WithFields(logrus.Fields{"folderPath": folderPath, "err": err}).Error("创建父文件夹异常")
			return nil, errors.Errorf("创建父文件夹异常: %+v", err)
		}
	}
	file, err := os.Create(filePath)
	if err != nil {
		logrus.WithContext(ctx).WithFields(logrus.Fields{"filePath": filePath, "err": err}).Error("创建文件异常")
		return nil, errors.Errorf("创建文件异常: %+v", err)
	}
	return file, nil
}

func readFile(ctx context.Context, filePath string) ([]byte, error) {
	file, err := os.Open(filePath)
	if err != nil {
		logrus.WithContext(ctx).WithFields(logrus.Fields{"filePath": filePath, "err": err}).Error("打开文件异常")
		return nil, errors.Errorf("打开文件异常: %+v", err)
	}
	defer file.Close()
	bytes, err := ioutil.ReadAll(file)
	if err != nil {
		logrus.WithContext(ctx).WithFields(logrus.Fields{"filePath": filePath, "err": err}).Error("读取文件异常")
		return nil, errors.Errorf("读取文件异常: %+v", err)
	}
	return bytes, nil
}

func ClearPath(ctx context.Context, fileOrFolderPath string) string {
	fileOrFolderPath = strings.ReplaceAll(fileOrFolderPath, "\\", "/")
	return path.Clean(fileOrFolderPath)
}

func GetFileMd5(ctx context.Context, filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		logrus.WithContext(ctx).WithFields(logrus.Fields{"filePath": filePath, "err": err}).Error("计算MD5打开文件异常")
		return "", errors.Errorf("计算MD5打开文件异常: %+v", err)
	}
	defer file.Close()
	md5 := md5.New()
	_, err = io.Copy(md5, file)
	if err != nil {
		logrus.WithContext(ctx).WithFields(logrus.Fields{"filePath": filePath, "err": err}).Error("计算MD5读取文件异常")
		return "", errors.Errorf("计算MD5读取文件异常: %+v", err)
	}
	return hex.EncodeToString(md5.Sum(nil)), nil
}
