package util

import (
	"context"
	"fmt"
	nested "github.com/antonfisher/nested-logrus-formatter"
	"github.com/gin-gonic/gin"
	"github.com/natefinch/lumberjack"
	"github.com/sirupsen/logrus"
	"io"
	"net/http"
	"os"
	"runtime"
	"strconv"
	"strings"
	"time"
)

const ReqIdKey = "reqid"
const LogIdKey = "logid"
const ServerNameKey = "sn"
const IpKey = "ip"
const CallerKey = "caller"

func InitDefaultLog() {
	InitLog(GetServerName(), "", 1, 100, 30, logrus.InfoLevel)
}

func CreateDefaultLog(filename string) *logrus.Logger {
	return CreateLog(GetServerName(), filename, 1, 100, 30, logrus.InfoLevel)
}

func InitLog(serverName, filename string, maxSize, maxBackups, maxAge int, level logrus.Level) {
	if serverName == "" {
		serverName = "log"
	}
	if filename == "" {
		filename = "log.log"
	}
	logrus.SetLevel(level)
	filename = fmt.Sprintf("log/%s/%s", serverName, filename)
	lumberJackLogger := &lumberjack.Logger{
		Filename:   filename,   //日志文件的位置
		MaxSize:    maxSize,    //在进行切割之前，日志文件的最大大小（以MB为单位）
		MaxBackups: maxBackups, //保留旧文件的最大个数
		MaxAge:     maxAge,     //保留旧文件的最大天数
		Compress:   false,      //是否压缩/归档旧文件
	}
	multiWriter := io.MultiWriter(os.Stdout, lumberJackLogger)
	logrus.SetOutput(multiWriter)
	logrus.SetFormatter(&nested.Formatter{
		HideKeys:        false, //显示 [fieldValue] 而不是 [fieldKey:fieldValue]
		NoFieldsColors:  true,  //仅将颜色应用于级别，默认为级别 + 字段
		TrimMessages:    true,  //修剪消息上的空格
		NoColors:        false, //禁用颜色
		TimestampFormat: time.RFC3339,
		FieldsOrder:     []string{LogIdKey, ServerNameKey, IpKey, CallerKey}, //字段排序，默认：字段按字母顺序排序
	})
	var hook paramHook
	hook.serverName = serverName
	logrus.AddHook(&hook)
}

func CreateLog(serverName, filename string, maxSize, maxBackups, maxAge int, level logrus.Level) *logrus.Logger {
	if serverName == "" {
		serverName = "log"
	}
	if filename == "" {
		filename = fmt.Sprintf("%s.log", GenStringId())
	}
	log := logrus.New()
	log.SetLevel(level)
	filename = fmt.Sprintf("log/%s/%s", serverName, filename)
	lumberJackLogger := &lumberjack.Logger{
		Filename:   filename,   //日志文件的位置
		MaxSize:    maxSize,    //在进行切割之前，日志文件的最大大小（以MB为单位）
		MaxBackups: maxBackups, //保留旧文件的最大个数
		MaxAge:     maxAge,     //保留旧文件的最大天数
		Compress:   false,      //是否压缩/归档旧文件
	}
	multiWriter := io.MultiWriter(os.Stdout, lumberJackLogger)
	log.SetOutput(multiWriter)
	log.SetFormatter(&nested.Formatter{
		HideKeys:        false, //显示 [fieldValue] 而不是 [fieldKey:fieldValue]
		NoFieldsColors:  true,  //仅将颜色应用于级别，默认为级别 + 字段
		TrimMessages:    true,  //修剪消息上的空格
		NoColors:        false, //禁用颜色
		TimestampFormat: time.RFC3339,
		FieldsOrder:     []string{LogIdKey, ServerNameKey, IpKey, CallerKey}, //字段排序，默认：字段按字母顺序排序
	})
	var hook paramHook
	hook.serverName = serverName
	log.AddHook(&hook)
	return log
}

type paramHook struct {
	serverName string
}

func (this *paramHook) Fire(entry *logrus.Entry) error {
	entry.Data[LogIdKey] = this.getLogId(entry)
	entry.Data[ServerNameKey] = this.serverName
	entry.Data[IpKey] = GetIp()
	entry.Data[CallerKey] = this.getCaller(entry)
	return nil
}
func (this *paramHook) getLogId(entry *logrus.Entry) int64 {
	if entry.Context == nil {
		return 0
	}
	return GetLogId(entry.Context)
}
func (this *paramHook) getCaller(entry *logrus.Entry) string {
	skip := 6
	var file string
	var line int
	ok := true
	for ok {
		_, file, line, ok = runtime.Caller(skip)
		skip++
		if !ok {
			break
		}
		if strings.Contains(file, "github.com/sirupsen/logrus") {
			continue
		}
		break
	}
	return fmt.Sprintf(`"%s:%d"`, file, line)
}
func (this *paramHook) Levels() []logrus.Level {
	return logrus.AllLevels
}

func GinLog(c *gin.Context) {
	startTime := time.Now()
	c.Next()
	endTime := time.Now()
	latencyTime := endTime.Sub(startTime)
	clientIP := c.ClientIP()
	method := c.Request.Method
	requestURI := c.Request.RequestURI
	status := c.Writer.Status()
	if status == http.StatusOK {
		logrus.WithContext(c).WithFields(logrus.Fields{"clientIP": clientIP, "method": method, "requestURI": requestURI, "status": status, "latencyTime": latencyTime}).Info()
	} else if status >= 500 {
		logrus.WithContext(c).WithFields(logrus.Fields{"clientIP": clientIP, "method": method, "requestURI": requestURI, "status": status, "latencyTime": latencyTime}).Error()
	} else {
		logrus.WithContext(c).WithFields(logrus.Fields{"clientIP": clientIP, "method": method, "requestURI": requestURI, "status": status, "latencyTime": latencyTime}).Warn()
	}
}

func GenLogId() int64 {
	return GenId()
}
func GetLogId(ctx context.Context) int64 {
	idP := GetCtxValue(ctx, LogIdKey)
	id, _ := idP.(int64)
	return id
}
func GetLogIdString(ctx context.Context) string {
	return strconv.FormatInt(GetLogId(ctx), 10)
}
func SetLogId(ctx context.Context) context.Context {
	id := GetLogId(ctx)
	if id <= 0 {
		id = GenLogId()
		ctx = SetCtxValue(ctx, LogIdKey, id)
	}
	return ctx
}
func ResetLogId(ctx context.Context) context.Context {
	id := GenLogId()
	ctx = SetCtxValue(ctx, LogIdKey, id)
	return ctx
}

func GenReqId() int64 {
	return GenId()
}
func GetReqId(ctx context.Context) int64 {
	idP := GetCtxValue(ctx, ReqIdKey)
	id, _ := idP.(int64)
	return id
}
func GetOrGenReqId(ctx context.Context) int64 {
	id := GetReqId(ctx)
	if id <= 0 {
		id = GenReqId()
	}
	return id
}
func GetOrGenReqIdString(ctx context.Context) string {
	return strconv.FormatInt(GetOrGenReqId(ctx), 10)
}
func SetReqId(ctx context.Context) context.Context {
	id := GetReqId(ctx)
	if id <= 0 {
		id = GenReqId()
		ctx = SetCtxValue(ctx, ReqIdKey, id)
	}
	return ctx
}
func ResetReqId(ctx context.Context) context.Context {
	id := GenReqId()
	ctx = SetCtxValue(ctx, ReqIdKey, id)
	return ctx
}
func RmReqId(ctx context.Context) context.Context {
	ctx = SetCtxValue(ctx, ReqIdKey, 0)
	return ctx
}
