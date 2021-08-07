package util

import (
	"context"
	"fmt"
	"github.com/sirupsen/logrus"
	"strconv"
	"strings"
	"time"
)

func CreateId() int64 {
	now := time.Now()
	str := now.Format(DateLayout_060102150405_0000000)
	str = str[:12] + str[13:]
	logId, _ := strconv.ParseInt(str, 10, 64)
	return logId
}

func GenGoLabel(ctx context.Context, code string, labels ...string) string {
	if code == "" {
		logrus.WithContext(ctx).WithFields(logrus.Fields{}).Warn("空代码片段")
		return code
	}
	type Param struct {
		Name  string
		Type  string
		Label string
		Note  string
	}
	lines := strings.Split(code, "\n")
	for i := range lines {
		line := lines[i]
		line = strings.Trim(line, " ")
		line = strings.Trim(line, "\t")
		if line == "" || strings.Contains(line, "{") || strings.Contains(line, "}") {
			continue
		}
		var param Param
		keys := strings.SplitN(line, "//", 2)
		if len(keys) > 1 {
			param.Note = keys[1]
		}
		line = keys[0]
		keys = strings.Split(line, " ")
		if len(keys) < 2 {
			logrus.WithContext(ctx).WithFields(logrus.Fields{}).Warn("有参数行字段少于两个")
			continue
		}
		for j := range keys {
			key := keys[j]
			key = strings.Trim(key, " ")
			key = strings.Trim(key, "\t")
			if key == "" {
				continue
			}
			if param.Name == "" {
				param.Name = key
			} else if param.Type == "" {
				param.Type = key
				break
			}
		}
		param.Name = strings.Trim(param.Name, " ")
		param.Name = strings.Trim(param.Name, "\t")
		param.Type = strings.Trim(param.Type, " ")
		param.Type = strings.Trim(param.Type, "\t")
		underscoreName := Hump2Underscore(param.Name)
		param.Label = fmt.Sprintf("`json:\"%s\"", underscoreName)
		labelMap := make(map[string]bool)
		labelMap["json"] = true
		for _, label := range labels {
			param.Label += fmt.Sprintf(" %s:\"%s\"", label, underscoreName)
		}
		param.Label += "`"
		lines[i] = fmt.Sprintf("\t%s %s %s //%s", param.Name, param.Type, param.Label, param.Note)
	}
	code = strings.Join(lines, "\n")
	return code
}

func Hump2Underscore(text string) string {
	for j := 'A'; j <= 'Z'; j++ {
		text = strings.ReplaceAll(text, fmt.Sprintf("%c", j), fmt.Sprintf("_%c", j+32))
	}
	if text[0] == '_' {
		text = text[1:]
	}
	return text
}
