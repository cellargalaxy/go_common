package util

import (
	"bufio"
	"context"
	"crypto/md5"
	"encoding/hex"
	"io"
	"io/fs"
	"io/ioutil"
	"os"
	"path"
	"reflect"
	"strings"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

func CloseIo(ctx context.Context, list ...io.Closer) {
	for i := range list {
		//list[i] == nil 只能挡住"空接口"，挡不住"非空接口里装了nil指针"(typed-nil)：
		//如 var f *excelize.File; CloseIo(ctx, f) 时接口本身非nil，
		//直接调 Close() 会解引用空指针而panic。
		//这正是 DeGzip、XlsxData2Strings 此前"先defer再判err"会崩的根因，
		//在此统一兜住，调用方才不必各自小心翼翼地调整defer顺序。
		if isNilIo(list[i]) {
			continue
		}
		err := list[i].Close()
		if err != nil {
			logrus.WithContext(ctx).WithFields(logrus.Fields{"err": err}).Error("IO流关闭异常")
		}
	}
}

// isNilIo 判断closer是否为nil，含"非空接口装了nil指针"的情形。
// 只对本身可为nil的类型(指针/map/切片/通道/函数/接口)取Value.IsNil，
// 其余类型(如值接收器的结构体)一律视为非nil，避免reflect对不支持的类型panic。
func isNilIo(closer io.Closer) bool {
	if closer == nil {
		return true
	}
	value := reflect.ValueOf(closer)
	switch value.Kind() {
	case reflect.Ptr, reflect.Map, reflect.Slice, reflect.Chan, reflect.Func, reflect.UnsafePointer, reflect.Interface:
		return value.IsNil()
	default:
		return false
	}
}

func ClearPath(fileOrFolderPath string) string {
	fileOrFolderPath = strings.ReplaceAll(fileOrFolderPath, "\\", "/")
	return path.Clean(fileOrFolderPath)
}

func ListFile(ctx context.Context, folderPath string) ([]fs.FileInfo, error) {
	fileInfo := GetPathInfo(ctx, folderPath)
	if fileInfo == nil {
		logrus.WithFields(logrus.Fields{"folderPath": folderPath}).Warn("罗列文件，文件夹不存在")
		return nil, nil
	}
	if !fileInfo.IsDir() {
		logrus.WithFields(logrus.Fields{"folderPath": folderPath}).Warn("罗列文件，不是文件夹")
		return nil, nil
	}
	files, err := ioutil.ReadDir(folderPath)
	if err != nil {
		logrus.WithFields(logrus.Fields{"folderPath": folderPath, "err": err}).Error("罗列文件，读取文件夹异常")
		return nil, errors.Errorf("罗列文件，读取文件夹异常: %+v", err)
	}
	return files, nil
}

func GetPathInfo(ctx context.Context, path string) os.FileInfo {
	fileInfo, err := os.Stat(path)
	if fileInfo == nil {
		return fileInfo
	}
	if err == nil || os.IsExist(err) {
		return fileInfo
	}
	logrus.WithContext(ctx).WithFields(logrus.Fields{"path": path, "err": err}).Error("查询文件信息异常")
	return nil
}
func GetFolderInfo(ctx context.Context, folderPath string) os.FileInfo {
	info := GetPathInfo(ctx, folderPath)
	if info == nil {
		return info
	}
	if info.IsDir() {
		return info
	}
	return nil
}
func GetFileInfo(ctx context.Context, filePath string) os.FileInfo {
	info := GetPathInfo(ctx, filePath)
	if info == nil {
		return info
	}
	if info.IsDir() {
		return nil
	}
	return info
}

func CreateFolderPath(ctx context.Context, folderPath string) error {
	err := os.MkdirAll(folderPath, 0750)
	if err != nil {
		logrus.WithContext(ctx).WithFields(logrus.Fields{"folderPath": folderPath, "err": err}).Error("创建文件夹异常")
		return errors.Errorf("创建文件夹异常: %+v", err)
	}
	return nil
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
		return nil, errors.Errorf("文件创建异常: %+v", err)
	}
	logrus.WithContext(ctx).WithFields(logrus.Fields{"filePath": filePath}).Info("文件创建完成")
	return file, nil
}

func openReadFile(ctx context.Context, filePath string) (*os.File, error) {
	file, err := os.OpenFile(filePath, os.O_RDONLY, 0600)
	if err != nil {
		logrus.WithContext(ctx).WithFields(logrus.Fields{"filePath": filePath, "err": err}).Error("文件打开异常")
		return nil, errors.Errorf("文件打开异常: %+v", err)
	}
	return file, nil
}
func OpenReadFile(ctx context.Context, filePath string) (*os.File, error) {
	fileInfo := GetPathInfo(ctx, filePath)
	if fileInfo == nil {
		return createFile(ctx, filePath)
	}
	if fileInfo.IsDir() {
		logrus.WithContext(ctx).WithFields(logrus.Fields{"filePath": filePath}).Error("该路径为文件夹，获取文件失败")
		return nil, errors.Errorf("该路径为文件夹，获取文件失败")
	}
	return openReadFile(ctx, filePath)
}

func openWriteFile(ctx context.Context, filePath string) (*os.File, error) {
	file, err := os.OpenFile(filePath, os.O_RDWR|os.O_TRUNC, 0600)
	if err != nil {
		logrus.WithContext(ctx).WithFields(logrus.Fields{"filePath": filePath, "err": err}).Error("文件打开异常")
		return nil, errors.Errorf("文件打开异常: %+v", err)
	}
	return file, nil
}
func OpenWriteFile(ctx context.Context, filePath string) (*os.File, error) {
	fileInfo := GetPathInfo(ctx, filePath)
	if fileInfo == nil {
		return createFile(ctx, filePath)
	}
	if fileInfo.IsDir() {
		logrus.WithContext(ctx).WithFields(logrus.Fields{"filePath": filePath}).Error("该路径为文件夹，获取文件失败")
		return nil, errors.Errorf("该路径为文件夹，获取文件失败")
	}
	return openWriteFile(ctx, filePath)
}

func WriteData2File(ctx context.Context, data []byte, filePath string) error {
	file, err := OpenWriteFile(ctx, filePath)
	if err != nil {
		return err
	}
	defer func(filePath string, file *os.File) {
		err := file.Close()
		if err != nil {
			logrus.WithContext(ctx).WithFields(logrus.Fields{"filePath": filePath, "err": err}).Error("文件关闭异常")
		}
	}(filePath, file)
	written, err := file.Write(data)
	if err != nil {
		logrus.WithContext(ctx).WithFields(logrus.Fields{"filePath": filePath, "err": err}).Error("文件写入异常")
		return errors.Errorf("文件写入异常: %+v", err)
	} else {
		logrus.WithContext(ctx).WithFields(logrus.Fields{"filePath": filePath, "written": written}).Info("文件写入完成")
	}
	return nil
}
func WriteString2File(ctx context.Context, text string, filePath string) error {
	return WriteData2File(ctx, []byte(text), filePath)
}
func WriteReader2File(ctx context.Context, reader io.Reader, filePath string) error {
	file, err := OpenWriteFile(ctx, filePath)
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
		return errors.Errorf("文件写入异常: %+v", err)
	} else {
		logrus.WithContext(ctx).WithFields(logrus.Fields{"filePath": filePath, "written": written}).Info("文件写入完成")
	}
	return nil
}

func ReadFile2Data(ctx context.Context, filePath string, defaultData []byte) ([]byte, error) {
	//文件不存在时返回默认值，既不报错也不创建文件，保持只读语义
	if GetPathInfo(ctx, filePath) == nil {
		logrus.WithContext(ctx).WithFields(logrus.Fields{"filePath": filePath}).Warn("文件不存在，返回默认值")
		return defaultData, nil
	}
	file, err := OpenReadFile(ctx, filePath)
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
		return nil, errors.Errorf("文件读取异常: %+v", err)
	} else {
		logrus.WithContext(ctx).WithFields(logrus.Fields{"filePath": filePath, "len(data)": len(data)}).Info("文件读取完成")
	}
	if len(data) == 0 {
		data = defaultData
	}
	return data, nil
}
func ReadFile2String(ctx context.Context, filePath string, defaultText string) (string, error) {
	data, err := ReadFile2Data(ctx, filePath, []byte(defaultText))
	if err != nil {
		return "", err
	}
	return string(data), nil
}
func ReadFile2Writer(ctx context.Context, filePath string, writer io.Writer, defaultData []byte) error {
	//文件不存在时写出默认值，保持与ReadFile2Data一致的只读语义
	if GetPathInfo(ctx, filePath) == nil {
		logrus.WithContext(ctx).WithFields(logrus.Fields{"filePath": filePath}).Warn("文件不存在，写出默认值")
		if len(defaultData) == 0 {
			return nil
		}
		written, err := writer.Write(defaultData)
		if err != nil {
			logrus.WithContext(ctx).WithFields(logrus.Fields{"filePath": filePath, "err": err}).Error("文件拷贝数据异常")
			return errors.Errorf("文件拷贝数据异常: %+v", err)
		}
		logrus.WithContext(ctx).WithFields(logrus.Fields{"filePath": filePath, "written": written}).Info("文件拷贝数据完成")
		return nil
	}
	file, err := OpenReadFile(ctx, filePath)
	if err != nil {
		return err
	}
	defer func(filePath string, file *os.File) {
		err := file.Close()
		if err != nil {
			logrus.WithContext(ctx).WithFields(logrus.Fields{"filePath": filePath, "err": err}).Error("文件关闭异常")
		}
	}(filePath, file)

	written, err := io.Copy(writer, file)
	if err != nil {
		logrus.WithContext(ctx).WithFields(logrus.Fields{"filePath": filePath, "err": err}).Error("文件拷贝数据异常")
		return errors.Errorf("文件拷贝数据异常: %+v", err)
	} else {
		logrus.WithContext(ctx).WithFields(logrus.Fields{"filePath": filePath, "written": written}).Info("文件拷贝数据完成")
	}

	if written == 0 {
		written, err := writer.Write(defaultData)
		if err != nil {
			logrus.WithContext(ctx).WithFields(logrus.Fields{"filePath": filePath, "err": err}).Error("文件拷贝数据异常")
			return errors.Errorf("文件拷贝数据异常: %+v", err)
		} else {
			logrus.WithContext(ctx).WithFields(logrus.Fields{"filePath": filePath, "written": written}).Info("文件拷贝数据完成")
		}
	}

	return nil
}

func GetFileMd5(ctx context.Context, filePath string) (string, error) {
	file, err := OpenReadFile(ctx, filePath)
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
		return "", errors.Errorf("文件计算MD5异常: %+v", err)
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}

func RemoveFile(ctx context.Context, filePath string) error {
	fileInfo := GetPathInfo(ctx, filePath)
	if fileInfo == nil {
		logrus.WithFields(logrus.Fields{"filePath": filePath}).Warn("删除文件，文件不存在")
		return nil
	}
	if fileInfo.IsDir() {
		files, err := ioutil.ReadDir(filePath)
		if err != nil {
			logrus.WithFields(logrus.Fields{"filePath": filePath, "err": err}).Error("删除文件，读取文件夹异常")
			return errors.Errorf("删除文件，读取文件夹异常: %+v", err)
		}
		if len(files) > 0 {
			logrus.WithFields(logrus.Fields{"filePath": filePath}).Error("删除文件，文件夹不为空")
			return errors.Errorf("删除文件，文件夹不为空")
		}
	}
	err := os.Remove(filePath)
	if err != nil {
		logrus.WithContext(ctx).WithFields(logrus.Fields{"filePath": filePath}).Error("删除文件，异常")
		return errors.Errorf("删除文件，异常: %+v", err)
	}
	return nil
}

func Read2LogByReader(ctx context.Context, reader *bufio.Reader, save bool) ([]string, error) {
	var lines []string
	//空行统一跳过：EOF分支与正常分支必须同一套标准，
	//否则"a\n\nb\n"会收集到空串、而"a\n\nb"（末尾无换行）不会，同一函数两种语义
	appendLine := func(line string) {
		line = strings.TrimSpace(line)
		if line == "" {
			return
		}
		logrus.WithFields(logrus.Fields{"line": line}).Info("流读取")
		if save {
			lines = append(lines, line)
		}
	}
	for {
		line, err := reader.ReadString('\n')
		if err == io.EOF {
			//EOF时可能带回最后一段无换行内容，需补入结果避免丢数据
			appendLine(line)
			logrus.WithFields(logrus.Fields{}).Info("流读取，完成")
			return lines, nil
		}
		if err != nil {
			logrus.WithFields(logrus.Fields{"err": err}).Error("流读取，异常")
			return lines, errors.Errorf("流读取，异常: %+v", err)
		}
		appendLine(line)
	}
}
