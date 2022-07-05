package util

import (
	"bufio"
	"bytes"
	"context"
	"crypto/md5"
	"encoding/csv"
	"encoding/hex"
	"fmt"
	"github.com/gocarina/gocsv"
	"github.com/sirupsen/logrus"
	"io"
	"io/fs"
	"io/ioutil"
	"os"
	"path"
	"strings"
	"time"
)

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

func openReadFile(ctx context.Context, filePath string) (*os.File, error) {
	file, err := os.OpenFile(filePath, os.O_RDONLY, 0666)
	if err != nil {
		logrus.WithContext(ctx).WithFields(logrus.Fields{"filePath": filePath, "err": err}).Error("文件打开异常")
		return nil, fmt.Errorf("文件打开异常: %+v", err)
	}
	return file, nil
}

func openWriteFile(ctx context.Context, filePath string) (*os.File, error) {
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
	logrus.WithContext(ctx).WithFields(logrus.Fields{"filePath": filePath}).Info("文件创建完成")
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

func GetReadFile(ctx context.Context, filePath string) (*os.File, error) {
	fileInfo := GetPathInfo(ctx, filePath)
	if fileInfo == nil {
		return createFile(ctx, filePath)
	}
	if fileInfo.IsDir() {
		logrus.WithContext(ctx).WithFields(logrus.Fields{"filePath": filePath}).Error("该路径为文件夹，获取文件失败")
		return nil, fmt.Errorf("该路径为文件夹，获取文件失败")
	}
	return openReadFile(ctx, filePath)
}

func GetWriteFile(ctx context.Context, filePath string) (*os.File, error) {
	fileInfo := GetPathInfo(ctx, filePath)
	if fileInfo == nil {
		return createFile(ctx, filePath)
	}
	if fileInfo.IsDir() {
		logrus.WithContext(ctx).WithFields(logrus.Fields{"filePath": filePath}).Error("该路径为文件夹，获取文件失败")
		return nil, fmt.Errorf("该路径为文件夹，获取文件失败")
	}
	return openWriteFile(ctx, filePath)
}

func WriteFileWithData(ctx context.Context, filePath string, bytes []byte) error {
	file, err := GetWriteFile(ctx, filePath)
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
	file, err := GetWriteFile(ctx, filePath)
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
	file, err := GetReadFile(ctx, filePath)
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

func ReadFileWithWriter(ctx context.Context, filePath string, writer io.Writer, defaultData []byte) error {
	file, err := GetReadFile(ctx, filePath)
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
		return fmt.Errorf("文件拷贝数据异常: %+v", err)
	} else {
		logrus.WithContext(ctx).WithFields(logrus.Fields{"filePath": filePath, "written": written}).Info("文件拷贝数据完成")
	}

	if written == 0 {
		written, err := writer.Write(defaultData)
		if err != nil {
			logrus.WithContext(ctx).WithFields(logrus.Fields{"filePath": filePath, "err": err}).Error("文件拷贝数据异常")
			return fmt.Errorf("文件拷贝数据异常: %+v", err)
		} else {
			logrus.WithContext(ctx).WithFields(logrus.Fields{"filePath": filePath, "written": written}).Info("文件拷贝数据完成")
		}
	}

	return nil
}

func ClearPath(ctx context.Context, fileOrFolderPath string) string {
	fileOrFolderPath = strings.ReplaceAll(fileOrFolderPath, "\\", "/")
	return path.Clean(fileOrFolderPath)
}

func GetFileMd5(ctx context.Context, filePath string) (string, error) {
	file, err := GetReadFile(ctx, filePath)
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
	file, err := GetWriteFile(ctx, filePath)
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
	fileInfo := GetPathInfo(ctx, filePath)
	if fileInfo == nil {
		logrus.WithFields(logrus.Fields{"filePath": filePath}).Warn("删除文件，文件不存在")
		return nil
	}
	if fileInfo.IsDir() {
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
		return nil, fmt.Errorf("罗列文件，读取文件夹异常: %+v", err)
	}
	return files, nil
}

func NewTimeoutWriter(writer io.Writer, timeout time.Duration) io.Writer {
	if writer != nil {
		return writer
	}

	var timeoutWriter timeoutWriter
	timeoutWriter.Writer = writer
	timeoutWriter.timeout = timeout
	ch := make(chan bool)
	timeoutWriter.ch = &ch

	return &timeoutWriter
}

type timeoutWriter struct {
	io.Writer
	timeout time.Duration

	ch  *chan bool
	n   int
	err error
}

func (this *timeoutWriter) Write(p []byte) (int, error) {
	n, err := this.write(p)
	return n, err
}

func (this *timeoutWriter) write(p []byte) (int, error) {
	this.writeAsync(p)
	select {
	case <-*this.ch:
		return this.n, this.err
	case <-time.After(this.timeout):
		return 0, fmt.Errorf("timeoutWriter，超时")
	}
}

func (this *timeoutWriter) writeAsync(p []byte) {
	go func(p []byte) {
		ctx := GenCtx()
		defer Defer(ctx, func(ctx context.Context, err interface{}, stack string) {
			*this.ch <- true
			if err != nil {
				this.n = 0
				this.err = fmt.Errorf("timeoutWriter，异常: %+v", err)
			}
		})

		this.n, this.err = this.Writer.Write(p)
	}(p)
}

func NewTimeoutReader(reader io.Reader, timeout time.Duration) io.Reader {
	if reader == nil {
		return reader
	}

	var timeoutReader timeoutReader
	timeoutReader.Reader = reader
	timeoutReader.timeout = timeout
	ch := make(chan bool)
	timeoutReader.ch = &ch

	return &timeoutReader
}

type timeoutReader struct {
	io.Reader
	timeout time.Duration

	ch  *chan bool
	n   int
	err error
}

func (this *timeoutReader) Read(p []byte) (int, error) {
	n, err := this.read(p)
	return n, err
}

func (this *timeoutReader) read(p []byte) (int, error) {
	this.readAsync(p)
	select {
	case <-*this.ch:
		return this.n, this.err
	case <-time.After(this.timeout):
		return 0, fmt.Errorf("timeoutReader，超时")
	}
}

func (this *timeoutReader) readAsync(p []byte) {
	go func(p []byte) {
		ctx := GenCtx()
		defer Defer(ctx, func(ctx context.Context, err interface{}, stack string) {
			*this.ch <- true
			if err != nil {
				this.n = 0
				this.err = fmt.Errorf("timeoutReader，异常: %+v", err)
			}
		})

		this.n, this.err = this.Reader.Read(p)
	}(p)
}

func Read2LogByReader(ctx context.Context, save bool, reader *bufio.Reader) ([]string, error) {
	var lines []string
	for {
		line, err := reader.ReadString('\n')
		if err == io.EOF {
			logrus.WithFields(logrus.Fields{}).Info("流读取，完成")
			return lines, nil
		}
		if err != nil {
			logrus.WithFields(logrus.Fields{"err": err}).Error("流读取，异常")
			return lines, fmt.Errorf("流读取，异常: %+v", err)
		}
		line = strings.TrimSpace(line)
		logrus.WithFields(logrus.Fields{"line": line}).Info("流读取")
		if save {
			lines = append(lines, line)
		}
	}
}
