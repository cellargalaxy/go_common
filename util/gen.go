package util

import (
	"context"
	"fmt"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"math/rand"
	"strings"
	"sync"
	"time"
)

var runes = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

func CreateRandString(n int) string {
	//n为负会让 make([]rune, n) 直接panic(makeslice: len out of range)，
	//语义上负长度与0长度等价，统一夹紧后返回空串
	if n <= 0 {
		return ""
	}
	b := make([]rune, n)
	for i := range b {
		b[i] = runes[rand.Intn(len(runes))]
	}
	return string(b)
}

func GenIdByTime(time time.Time) int64 {
	//必须与ParseStringId的解析时区E8Loc保持一致，否则非UTC+8环境下ID无法往返解析
	str := time.In(E8Loc).Format(DateLayout_060102150405_0000000)
	str = str[:12] + str[13:]
	return String2Int[int64](str)
}

// lastIdMicro 记录上一次GenId发出的ID所对应的Unix微秒时刻，用于保证ID唯一。
// ID格式只到微秒(见DateLayout_060102150405_0000000的.000000)，而time.Now()的
// 实测最小步进约几十纳秒，因此同一微秒内的多次调用会拿到完全相同的ID：
// 实测顺序调用2万次仅得约2700个不同值(碰撞率86%)，20并发下碰撞率约97%。
// GenId同时是GenLogId与GenReqId的底层实现，而ValidateGin用reqId做防重放校验
// (existReqId命中即拒绝请求)，ID重复会让合法请求被误判为重放而拒绝，故必须保证唯一。
var lastIdMicro struct {
	sync.Mutex
	micro int64
}

// nextIdMicro 返回严格递增的Unix微秒值：正常返回当前时刻，
// 若当前时刻未超过上次发号时刻(同一微秒内连续调用，或系统时钟回拨)，则取上次值+1。
// 递增在"微秒时刻"上进行而非直接对ID整数+1：ID是060102150405.000000的拼接结果，
// 直接对整数+1会在边界产生非法值(如秒位变成60，ParseId报second out of range)，
// 而在时间域上+1微秒可让秒/分/时/日/年逐级正常进位。
func nextIdMicro(now time.Time) int64 {
	micro := now.UnixMicro()
	lastIdMicro.Lock()
	defer lastIdMicro.Unlock()
	if micro <= lastIdMicro.micro {
		micro = lastIdMicro.micro + 1
	}
	lastIdMicro.micro = micro
	return micro
}

func GenId() int64 {
	return GenIdByTime(time.UnixMicro(nextIdMicro(time.Now())))
}
func GenStringId() string {
	return Int2String(GenId())
}
func ParseId(ctx context.Context, id int64) (time.Time, error) {
	return ParseStringId(ctx, Int2String(id))
}
func ParseStringId(ctx context.Context, id string) (time.Time, error) {
	//ID是"年份后两位+月日时分秒+微秒"的定长18位数字串，年份后两位为0x时首位是0
	//(2000~2009、2100~2109等)，一旦经 int64 承载再 Int2String 输出，前导零会丢失：
	//实测 GenIdByTime(2006-01-02 15:04:05) 得17位ID，被本函数按"非法长度"拒绝，
	//即 GenId 生成的ID自己的反函数解析不了，故先左补零还原到18位再解析。
	//补零不会放宽校验：月/日/时若为00，time.ParseInLocation 仍会报错（实测全部拒绝）。
	if 0 < len(id) && len(id) < 18 {
		id = strings.Repeat("0", 18-len(id)) + id
	}
	if len(id) != 18 {
		logrus.WithContext(ctx).WithFields(logrus.Fields{}).Warn("解析ID，非法长度ID")
		return time.Time{}, errors.Errorf("解析ID，非法长度ID")
	}
	id = id[:12] + "." + id[12:]
	return ParseStr2Time(ctx, DateLayout_060102150405_0000000, id, E8Loc)
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
		line = strings.TrimSpace(line)
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
			key = strings.TrimSpace(key)
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
		param.Name = strings.TrimSpace(param.Name)
		param.Type = strings.TrimSpace(param.Type)
		underscoreName := Hump2Underscore(param.Name)
		param.Label = fmt.Sprintf("`json:\"%s\"", underscoreName)
		//labelMap 用于标签去重：json 由上一行固定写入，
		//若 labels 再传入 json（或同名标签重复传入），拼出的
		//`json:"x" json:"x"` 是重复键，会让后续 reflect 取标签的行为依赖实现细节。
		//此前该map只写不读，等于去重逻辑失效，实测 GenGoLabel(code,"json") 真会输出重复标签
		labelMap := make(map[string]bool)
		labelMap["json"] = true
		for _, label := range labels {
			label = strings.TrimSpace(label)
			if label == "" || labelMap[label] {
				continue
			}
			labelMap[label] = true
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
		line = strings.TrimSpace(line)
		if line == "" || strings.Contains(line, "}") {
			continue
		}
		if strings.Contains(line, "{") {
			line = strings.ReplaceAll(line, "type", "")
			line = strings.ReplaceAll(line, "struct", "")
			line = strings.ReplaceAll(line, "{", "")
			line = strings.TrimSpace(line)
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
			key = strings.TrimSpace(key)
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
		param.Name = strings.TrimSpace(param.Name)
		param.Type = strings.TrimSpace(param.Type)
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
		//未知类型不能把Go类型名原样当作默认值输出，
		//那会生成 "`flag` bool NOT NULL bool" 这种语法必然非法的SQL。
		//此处返回空串表示不带DEFAULT子句，列类型仍由getBdType原样透出以便人工核对。
		return ""
	}
}
