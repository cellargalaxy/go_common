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

func GenModel2Sql(ctx context.Context, code string) string {
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
	var modelName string
	var params []Param
	for i := range lines {
		line := lines[i]
		line = strings.Trim(line, " ")
		line = strings.Trim(line, "\t")
		if line == "" || strings.Contains(line, "}") {
			continue
		}
		if strings.Contains(line, "{") {
			line = strings.ReplaceAll(line, "type", "")
			line = strings.ReplaceAll(line, "struct", "")
			line = strings.ReplaceAll(line, "{", "")
			line = strings.Trim(line, " ")
			line = strings.Trim(line, "\t")
			modelName = line
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
		params = append(params, param)
	}
	lines = make([]string, 0)
	lines = append(lines, fmt.Sprintf("CREATE TABLE `%s`", Hump2Underscore(modelName)))
	lines = append(lines, "(")
	lines = append(lines, "    `id`            int(11)      NOT NULL AUTO_INCREMENT,")
	for i := range params {
		lines = append(lines, fmt.Sprintf("`%s` %s NOT NULL %s COMMENT '%s',", Hump2Underscore(params[i].Name), getBdType(params[i].Type), getBdDefaultValue(params[i].Type), params[i].Note))
	}
	lines = append(lines, "    `created_at`             datetime     NOT NULL,")
	lines = append(lines, "    `updated_at`             datetime     NOT NULL,")
	lines = append(lines, "    PRIMARY KEY (`id`)")
	lines = append(lines, ") ENGINE = InnoDB")
	lines = append(lines, "  DEFAULT CHARSET = utf8mb4")
	lines = append(lines, "  COLLATE = utf8mb4_unicode_ci;")
	code = strings.Join(lines, "\n")
	return code
}

func getBdType(goType string) string {
	switch goType {
	case "int":
		return "int(11)"
	case "int64":
		return "bigint(20)"
	case "string":
		return "varchar(255)"
	case "time.Time":
		return "datetime"
	case "float32":
		return "float"
	case "float64":
		return "double"
	default:
		return goType
	}
}
func getBdDefaultValue(goType string) string {
	switch goType {
	case "int":
		return "DEFAULT 0"
	case "int64":
		return "DEFAULT 0"
	case "string":
		return "DEFAULT ''"
	case "time.Time":
		return ""
	case "float32":
		return "DEFAULT 0"
	case "float64":
		return "DEFAULT 0"
	default:
		return goType
	}
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