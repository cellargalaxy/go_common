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
const CallerKey = "caller"

func InitLog(projectName string) {
	filename := fmt.Sprintf("log/%s.log", projectName)
	lumberJackLogger := &lumberjack.Logger{
		Filename:   filename, //日志文件的位置
		MaxSize:    1,        //在进行切割之前，日志文件的最大大小（以MB为单位）
		MaxBackups: 100,      //保留旧文件的最大个数
		MaxAge:     30,       //保留旧文件的最大天数
		Compress:   false,    //是否压缩/归档旧文件
	}
	multiWriter := io.MultiWriter(os.Stdout, lumberJackLogger)
	logrus.SetOutput(multiWriter)
	logrus.SetFormatter(&nested.Formatter{
		HideKeys:        false, //显示 [fieldValue] 而不是 [fieldKey:fieldValue]
		NoFieldsColors:  true,  //仅将颜色应用于级别，默认为级别 + 字段
		TrimMessages:    true,  //修剪消息上的空格
		NoColors:        false, //禁用颜色
		TimestampFormat: time.RFC3339,
		FieldsOrder:     []string{LogIdKey, CallerKey}, //字段排序，默认：字段按字母顺序排序
	})
	logrus.AddHook(LogIdHook{})
	logrus.AddHook(CallerHook{})
}

type LogIdHook struct {
}

func (this LogIdHook) Fire(entry *logrus.Entry) error {
	if entry.Context == nil {
		return nil
	}
	logId := GetLogId(entry.Context)
	entry.Data[LogIdKey] = logId
	return nil
}
func (this LogIdHook) Levels() []logrus.Level {
	return logrus.AllLevels
}

type CallerHook struct {
}

func (this CallerHook) Fire(entry *logrus.Entry) error {
	skip := 6
	var file string
	var line int
	ok := true
	for ok {
		_, file, line, ok = runtime.Caller(skip)
		//fmt.Println(skip, file, line)
		skip++
		if !ok {
			break
		}
		if strings.Contains(file, "github.com/sirupsen/logrus") {
			continue
		}
		break
	}
	entry.Data[CallerKey] = fmt.Sprintf(`"%s:%d"`, file, line)
	return nil
}
func (this CallerHook) Levels() []logrus.Level {
	return logrus.AllLevels
}

func GetLogId(ctx context.Context) int64 {
	logIdP := ctx.Value(LogIdKey)
	logId, _ := logIdP.(int64)
	return logId
}
func SetLogId(ctx context.Context) context.Context {
	logIdP := ctx.Value(LogIdKey)
	logId, ok := logIdP.(int64)
	if !ok {
		logId = CreateLogId()
		ctx = context.WithValue(ctx, LogIdKey, logId)
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
	c.Header(LogIdKey, fmt.Sprint(logId))
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
