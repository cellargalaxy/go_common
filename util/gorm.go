package util

import (
	"context"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"strings"
	"time"
)

const (
	DefaultSqlLen = 512
)

func NewDefaultGormLog() logger.Interface {
	return GormLog{handlers: []GormLogHandler{NewGormLogHandler()}}
}
func NewGormLog(handlers ...GormLogHandler) logger.Interface {
	return GormLog{handlers: handlers}
}

type GormLog struct {
	handlers []GormLogHandler
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
	sql, _ := fc()
	for i := range this.handlers {
		if this.handlers[i] == nil {
			continue
		}
		this.handlers[i].Handler(ctx, begin, sql, err)
	}
}

func NewGormLogHandler() GormLogHandler {
	return NewGormSqlHandler([]error{gorm.ErrRecordNotFound}, DefaultSqlLen, false, true, true, false, true)
}

type GormLogHandler interface {
	Handler(ctx context.Context, begin time.Time, sql string, err error)
}

func NewGormSqlHandler(ignoreErrs []error, sqlLen int, insertShow, deleteShow, selectShow, updateShow, otherShow bool) GormSqlHandler {
	return GormSqlHandler{IgnoreErrs: ignoreErrs, SqlLen: sqlLen, InsertShow: insertShow, DeleteShow: deleteShow, SelectShow: selectShow, UpdateShow: updateShow, OtherShow: otherShow}
}

type GormSqlHandler struct {
	IgnoreErrs []error
	SqlLen     int
	InsertShow bool
	DeleteShow bool
	SelectShow bool
	UpdateShow bool
	OtherShow  bool
}

func (this GormSqlHandler) Handler(ctx context.Context, begin time.Time, sql string, err error) {
	elapsed := time.Since(begin)
	if this.SqlLen > 0 && this.SqlLen < len(sql) {
		sql = sql[:this.SqlLen]
	}
	if err != nil {
		ignore := false
		for i := range this.IgnoreErrs {
			if errors.Is(err, this.IgnoreErrs[i]) {
				ignore = true
				break
			}
		}
		if !ignore {
			logrus.WithContext(ctx).WithFields(logrus.Fields{"err": err, "elapsed": elapsed, "sql": sql}).Error()
			return
		}
	}
	if strings.HasPrefix(sql, "INSERT") {
		if this.InsertShow {
			logrus.WithContext(ctx).WithFields(logrus.Fields{"elapsed": elapsed, "sql": sql}).Info()
		}
	} else if strings.HasPrefix(sql, "DELETE") {
		if this.DeleteShow {
			logrus.WithContext(ctx).WithFields(logrus.Fields{"elapsed": elapsed, "sql": sql}).Info()
		}
	} else if strings.HasPrefix(sql, "SELECT") {
		if this.SelectShow {
			logrus.WithContext(ctx).WithFields(logrus.Fields{"elapsed": elapsed, "sql": sql}).Info()
		}
	} else if strings.HasPrefix(sql, "UPDATE") {
		if this.UpdateShow {
			logrus.WithContext(ctx).WithFields(logrus.Fields{"elapsed": elapsed, "sql": sql}).Info()
		}
	} else if this.OtherShow {
		logrus.WithContext(ctx).WithFields(logrus.Fields{"elapsed": elapsed, "sql": sql}).Info()
	}
}
