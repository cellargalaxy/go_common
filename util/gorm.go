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
	DefaultSqlLen = 256
)

type GormLogHandle interface {
	Handle(ctx context.Context, begin time.Time, sql string, err error)
}

func NewGormLog(handles ...GormLogHandle) logger.Interface {
	return GormLog{handles: handles}
}

type GormLog struct {
	handles []GormLogHandle
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
	for i := range this.handles {
		if this.handles[i] == nil {
			continue
		}
		this.handles[i].Handle(ctx, begin, sql, err)
	}
}

func NewDefaultGormSqlHandle() GormLogHandle {
	return NewGormSqlHandle([]error{gorm.ErrRecordNotFound}, DefaultSqlLen, false, true, true, false, true)
}

func NewGormSqlHandle(ignoreErrs []error, sqlLen int, insertShow, deleteShow, selectShow, updateShow, otherShow bool) GormLogHandle {
	return GormSqlHandle{IgnoreErrs: ignoreErrs, SqlLen: sqlLen, InsertShow: insertShow, DeleteShow: deleteShow, SelectShow: selectShow, UpdateShow: updateShow, OtherShow: otherShow}
}

type GormSqlHandle struct {
	IgnoreErrs []error
	SqlLen     int
	InsertShow bool
	DeleteShow bool
	SelectShow bool
	UpdateShow bool
	OtherShow  bool
}

func (this GormSqlHandle) Handle(ctx context.Context, begin time.Time, sql string, err error) {
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
