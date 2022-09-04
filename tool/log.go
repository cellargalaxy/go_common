package tool

import (
	"context"
	"fmt"
	"github.com/cellargalaxy/go_common/util"
	"github.com/sirupsen/logrus"
	"sort"
	"strings"
)

func Log2Csv(ctx context.Context, logPath, csvPath string) error {
	var err error
	content := logPath
	if strings.HasSuffix(logPath, "log") {
		fileInfo := util.GetFileInfo(ctx, logPath)
		if fileInfo == nil {
			logrus.WithContext(ctx).WithFields(logrus.Fields{"logPath": logPath}).Error("转换日志，文件不存在")
			return fmt.Errorf("转换日志，文件不存在")
		}
		content, err = util.ReadFileWithString(ctx, logPath, "")
		if err != nil {
			return err
		}
	}
	lines := strings.Split(content, "\n")
	list := make([][]string, 0, len(lines))
	for i := range lines {
		lines[i] = strings.ReplaceAll(lines[i], "\u001B[36m", "")
		lines[i] = strings.ReplaceAll(lines[i], "\u001B[33m", "")
		lines[i] = strings.ReplaceAll(lines[i], "\u001B[31m", "")
		lines[i] = strings.ReplaceAll(lines[i], "\u001B[0m", "")
		if !strings.HasPrefix(lines[i], "20") {
			continue
		}
		if strings.HasSuffix(lines[i], "[running]:") {
			continue
		}
		date := lines[i][:25]
		lines[i] = lines[i][27:]
		params := strings.Split(lines[i], "] [")
		text := strings.Split(params[len(params)-1], "] ")[1]
		params[len(params)-1] = strings.Split(params[len(params)-1], "] ")[0]
		object := []string{date, params[0], params[1], params[2], params[3], params[4], text}
		params = params[5:]
		sort.Sort(logs(params))
		object = append(object, params...)
		list = append(list, object)
	}
	return util.WriteCsv2DataByFile(ctx, list, csvPath)
}

type logs []string

func (this logs) Len() int {
	return len(this)
}

func (this logs) Swap(i, j int) {
	this[i], this[j] = this[j], this[i]
}

func (this logs) Less(i, j int) bool {
	return len(this[i]) < len(this[j])
}
