package util

import (
	"context"
	"errors"
	"fmt"
	nested "github.com/antonfisher/nested-logrus-formatter"
	"github.com/gin-gonic/gin"
	"github.com/natefinch/lumberjack"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"io"
	"net/http"
	"os"
	"runtime"
	"strconv"
	"strings"
	"time"
)

const LogIdKey = "logid"
const ServerNameKey = "sn"
const IpKey = "ip"
const CallerKey = "caller"

func InitDefaultLog() {
	InitLog(GetServerName(), 1, 100, 30, logrus.InfoLevel)
}

func CreateDefaultLog() *logrus.Logger {
	return CreateLog(GetServerName(), 1, 100, 30, logrus.InfoLevel)
}

func InitLog(serverName string, maxSize, maxBackups, maxAge int, level logrus.Level) {
	if serverName == "" {
		serverName = "log"
	}
	logrus.SetLevel(level)
	filename := fmt.Sprintf("log/%s/log.log", serverName)
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
	hook.flushIpAsync()
	logrus.AddHook(&hook)
}

func CreateLog(serverName string, maxSize, maxBackups, maxAge int, level logrus.Level) *logrus.Logger {
	if serverName == "" {
		serverName = "log"
	}
	log := logrus.New()
	log.SetLevel(level)
	filename := fmt.Sprintf("log/%s/log.log", serverName)
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
	hook.flushIpAsync()
	log.AddHook(&hook)
	return log
}

type paramHook struct {
	serverName string
	ip         string
}

func (this *paramHook) Fire(entry *logrus.Entry) error {
	entry.Data[LogIdKey] = this.getLogId(entry)
	entry.Data[ServerNameKey] = this.serverName
	entry.Data[IpKey] = this.ip
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
func (this *paramHook) flushIpAsync() {
	go func() {
		ctx := CreateLogCtx()
		defer Defer(ctx, func(ctx context.Context, err interface{}, stack string) {
			this.flushIpAsync()
		})

		for {
			this.FlushIp()
			time.Sleep(time.Hour)
		}
	}()
}
func (this *paramHook) FlushIp() {
	for i := 0; i < 5; i++ {
		ip := HttpGetIp()
		if ip != "" {
			this.ip = ip
			return
		}
	}
}
func (this *paramHook) Levels() []logrus.Level {
	return logrus.AllLevels
}

func GetLogId(ctx context.Context) int64 {
	logIdP := GetCtxValue(ctx, LogIdKey)
	logId, _ := logIdP.(int64)
	return logId
}
func SetLogId(ctx context.Context) context.Context {
	logIdP := GetCtxValue(ctx, LogIdKey)
	logId, ok := logIdP.(int64)
	if !ok {
		logId = CreateLogId()
		ctx = SetCtxValue(ctx, LogIdKey, logId)
	}
	return ctx
}
func CreateLogId() int64 {
	return GenId()
}
func CreateLogCtx() context.Context {
	ctx := context.Background()
	return SetLogId(ctx)
}
func GinLogId(c *gin.Context) {
	logId := GetLogId(c)
	if logId <= 0 {
		logIdString := c.Request.Header.Get(LogIdKey)
		logId, _ = strconv.ParseInt(logIdString, 10, 64)
	}
	if logId <= 0 {
		logId = CreateLogId()
	}
	c.Set(LogIdKey, logId)
	c.Header(LogIdKey, strconv.Itoa(int(logId)))
	c.Next()
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
		logrus.WithContext(c).WithFields(logrus.Fields{"clientIP": clientIP, "method": method, "requestURI": requestURI, "status": status, "latencyTime": latencyTime}).Info("")
	} else if status >= 500 {
		logrus.WithContext(c).WithFields(logrus.Fields{"clientIP": clientIP, "method": method, "requestURI": requestURI, "status": status, "latencyTime": latencyTime}).Error("")
	} else {
		logrus.WithContext(c).WithFields(logrus.Fields{"clientIP": clientIP, "method": method, "requestURI": requestURI, "status": status, "latencyTime": latencyTime}).Warn("")
	}
}

type GormLog struct {
	ShowSql    bool
	IgnoreErrs []error
}

func (this GormLog) LogMode(logger.LogLevel) logger.Interface {
	return this
}
func (this GormLog) Info(ctx context.Context, s string, args ...interface{}) {
	logrus.WithContext(ctx).Infof(s, args)
}
func (this GormLog) Warn(ctx context.Context, s string, args ...interface{}) {
	logrus.WithContext(ctx).Warnf(s, args)
}
func (this GormLog) Error(ctx context.Context, s string, args ...interface{}) {
	logrus.WithContext(ctx).Errorf(s, args)
}
func (this GormLog) Trace(ctx context.Context, begin time.Time, fc func() (string, int64), err error) {
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		ignore := false
		for i := range this.IgnoreErrs {
			if errors.Is(err, this.IgnoreErrs[i]) {
				ignore = true
				break
			}
		}
		if !ignore {
			elapsed := time.Since(begin)
			sql, _ := fc()
			fields := logrus.Fields{"err": err, "elapsed": elapsed, "sql": sql}
			logrus.WithContext(ctx).WithFields(fields).Error()
			return
		}
	}
	if this.ShowSql {
		elapsed := time.Since(begin)
		sql, _ := fc()
		fields := logrus.Fields{"elapsed": elapsed, "sql": sql}
		logrus.WithContext(ctx).WithFields(fields).Info()
		return
	}
}
