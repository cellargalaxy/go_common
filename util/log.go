package util

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"runtime"
	"strings"
	"time"

	nested "github.com/antonfisher/nested-logrus-formatter"
	"github.com/gin-gonic/gin"
	"github.com/natefinch/lumberjack"
	"github.com/sirupsen/logrus"
)

const ReqIdKey = "reqid"
const LogIdKey = "logid"
const ServerNameKey = "sn"
const IpKey = "ip"
const CallerKey = "caller"

func initLog(serverName, filename string, maxSize, maxBackups, maxAge int, level logrus.Level) {
	if serverName == "" {
		serverName = GenStrId()
	}
	if filename == "" {
		filename = "log.log"
	}
	filename = fmt.Sprintf("log/%s/%s", serverName, filename)
	lumberJackLogger := &lumberjack.Logger{
		Filename:   filename,   //日志文件的位置
		MaxSize:    maxSize,    //在进行切割之前，日志文件的最大大小（以MB为单位）
		MaxBackups: maxBackups, //保留旧文件的最大个数
		MaxAge:     maxAge,     //保留旧文件的最大天数
		Compress:   false,      //是否压缩/归档旧文件
	}
	multiWriter := io.MultiWriter(os.Stdout, lumberJackLogger)

	logrus.SetLevel(level)
	logrus.SetOutput(multiWriter)
	logrus.SetFormatter(&nested.Formatter{
		HideKeys:        false, //显示 [fieldValue] 而不是 [fieldKey:fieldValue]
		NoFieldsColors:  true,  //仅将颜色应用于级别，默认为级别 + 字段
		TrimMessages:    true,  //修剪消息上的空格
		NoColors:        false, //禁用颜色
		TimestampFormat: time.RFC3339,
		FieldsOrder:     []string{LogIdKey, ServerNameKey, IpKey, CallerKey}, //字段排序，默认：字段按字母顺序排序
	})

	var hook LogrusHook
	hook.serverName = serverName
	logrus.AddHook(&hook)
}

type LogrusHook struct {
	serverName string
}

func (this *LogrusHook) Levels() []logrus.Level {
	return logrus.AllLevels
}
func (this *LogrusHook) Fire(entry *logrus.Entry) error {
	entry.Data[LogIdKey] = this.getLogId(entry)
	entry.Data[ServerNameKey] = this.serverName
	entry.Data[IpKey] = GetIP()
	entry.Data[CallerKey] = this.getCaller(entry)
	return nil
}
func (this *LogrusHook) getLogId(entry *logrus.Entry) int64 {
	if entry.Context == nil {
		return 0
	}
	return GetLogId(entry.Context)
}
func (this *LogrusHook) getCaller(entry *logrus.Entry) string {
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
	file = ClearPath(file)
	files := strings.Split(file, "/")
	for i := range files {
		files[i] = strings.Split(files[i], "@")[0]
	}
	file = strings.Join(files, "/")
	return fmt.Sprintf(`"%s:%d"`, file, line)
}

func GinLog(c *gin.Context) {
	startTime := time.Now()
	c.Next()
	consume := time.Now().Sub(startTime)
	ip := c.ClientIP()
	method := c.Request.Method
	uri := c.Request.RequestURI
	status := c.Writer.Status()
	if status == http.StatusOK {
		logrus.WithContext(c).WithFields(logrus.Fields{"ip": ip, "method": method, "uri": uri, "status": status, "consume": consume}).Info()
	} else if status >= 500 {
		logrus.WithContext(c).WithFields(logrus.Fields{"ip": ip, "method": method, "uri": uri, "status": status, "consume": consume}).Error()
	} else {
		logrus.WithContext(c).WithFields(logrus.Fields{"ip": ip, "method": method, "uri": uri, "status": status, "consume": consume}).Warn()
	}
}

func GetLogId(ctx context.Context) int64 {
	return GetCtxValue[int64](ctx, LogIdKey)
}
func SetLogId(ctx context.Context) context.Context {
	if 0 < GetLogId(ctx) {
		return ctx
	}
	return SetCtxValue(ctx, LogIdKey, GenId())
}

// Deprecated: Watch out for memory leaks.
func ReSetLogId(ctx context.Context) context.Context {
	return SetCtxValue(ctx, LogIdKey, GenId())
}

func GetReqId(ctx context.Context) int64 {
	return GetCtxValue[int64](ctx, ReqIdKey)
}
func GetOrGenReqId(ctx context.Context) int64 {
	id := GetReqId(ctx)
	if id <= 0 {
		id = GenId()
	}
	return id
}
func SetReqId(ctx context.Context) context.Context {
	if 0 < GetReqId(ctx) {
		return ctx
	}
	return SetCtxValue(ctx, ReqIdKey, GenId())
}

// Deprecated: Watch out for memory leaks.
func ReSetReqId(ctx context.Context) context.Context {
	id := GenId()
	ctx = SetCtxValue(ctx, ReqIdKey, id)
	return ctx
}

// Deprecated: Watch out for memory leaks.
func RmReqId(ctx context.Context) context.Context {
	ctx = SetCtxValue(ctx, ReqIdKey, 0)
	return ctx
}
