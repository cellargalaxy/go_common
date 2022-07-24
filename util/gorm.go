package util

import (
	"context"
	"errors"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm/logger"
	"strings"
	"time"
)

type GormLog struct {
	IgnoreErrs []error
	InsertShow bool
	DeleteShow bool
	SelectShow bool
	UpdateShow bool
	OtherShow  bool
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
	elapsed := time.Since(begin)
	sql, _ := fc()
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
	if this.InsertShow && strings.HasPrefix(sql, "INSERT") {
		logrus.WithContext(ctx).WithFields(logrus.Fields{"elapsed": elapsed, "sql": sql}).Error()
	} else if this.DeleteShow && strings.HasPrefix(sql, "DELETE") {
		logrus.WithContext(ctx).WithFields(logrus.Fields{"elapsed": elapsed, "sql": sql}).Error()
	} else if this.SelectShow && strings.HasPrefix(sql, "SELECT") {
		logrus.WithContext(ctx).WithFields(logrus.Fields{"elapsed": elapsed, "sql": sql}).Error()
	} else if this.UpdateShow && strings.HasPrefix(sql, "UPDATE") {
		logrus.WithContext(ctx).WithFields(logrus.Fields{"elapsed": elapsed, "sql": sql}).Error()
	} else if this.OtherShow {
		logrus.WithContext(ctx).WithFields(logrus.Fields{"elapsed": elapsed, "sql": sql}).Error()
	}
}
