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

func NewDefaultGormLog() GormLog {
	return NewGormLog([]error{gorm.ErrRecordNotFound}, DefaultSqlLen, false, true, true, false, true)
}
func NewGormLog(ignoreErrs []error, sqlLen int, insertShow, deleteShow, selectShow, updateShow, otherShow bool) GormLog {
	return GormLog{IgnoreErrs: ignoreErrs, SqlLen: sqlLen, InsertShow: insertShow, DeleteShow: deleteShow, SelectShow: selectShow, UpdateShow: updateShow, OtherShow: otherShow}
}

type GormLog struct {
	IgnoreErrs []error
	SqlLen     int
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
	logrus.WithContext(ctx).Infof(s, args...)
}
func (this GormLog) Warn(ctx context.Context, s string, args ...interface{}) {
	logrus.WithContext(ctx).Warnf(s, args...)
}
func (this GormLog) Error(ctx context.Context, s string, args ...interface{}) {
	logrus.WithContext(ctx).Errorf(s, args...)
}
func (this GormLog) Trace(ctx context.Context, begin time.Time, fc func() (string, int64), err error) {
	sql, _ := fc()
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

type GormObject interface {
	TableName() string
}
type GormInquiry interface {
	GormObject
	GetOffset() int
	GetLimit() int
}
type GormHandler[Inquiry GormInquiry] interface {
	GetName(ctx context.Context) string
	GetDb(ctx context.Context, where *gorm.DB) *gorm.DB
	Where(ctx context.Context, where *gorm.DB, inquiry Inquiry) *gorm.DB
}
type GormService[Object GormObject, Inquiry GormInquiry] struct {
	GormHandler[Inquiry]
}

func (this *GormService[Object, Inquiry]) Insert(ctx context.Context, object ...*Object) ([]*Object, error) {
	if len(object) == 0 {
		logrus.WithContext(ctx).WithFields(logrus.Fields{}).Warnf("插入%s，为空", this.GetName(ctx))
		return object, nil
	}
	where := this.GetDb(ctx, nil)
	err := where.Create(&object).Error
	if err != nil {
		logrus.WithContext(ctx).WithFields(logrus.Fields{"err": err}).Errorf("插入%s，异常", this.GetName(ctx))
		return object, errors.Errorf("插入%s，异常: %+v", this.GetName(ctx), err)
	}
	logrus.WithContext(ctx).WithFields(logrus.Fields{}).Infof("插入%s，完成", this.GetName(ctx))
	return object, nil
}
func (this *GormService[Object, Inquiry]) Delete(ctx context.Context, inquiry Inquiry) error {
	var where *gorm.DB
	where = this.Where(ctx, where, inquiry)
	if where == nil {
		logrus.WithContext(ctx).WithFields(logrus.Fields{"inquiry": inquiry}).Warnf("删除%s，条件为空", this.GetName(ctx))
		return errors.Errorf("删除%s，条件为空", this.GetName(ctx))
	}
	err := where.Delete(&inquiry).Error
	if err != nil {
		logrus.WithContext(ctx).WithFields(logrus.Fields{"err": err}).Errorf("删除%s，异常", this.GetName(ctx))
		return errors.Errorf("删除%s，异常: %+v", this.GetName(ctx), err)
	}
	logrus.WithContext(ctx).WithFields(logrus.Fields{}).Infof("删除%s，完成", this.GetName(ctx))
	return nil
}
func (this *GormService[Object, Inquiry]) Update(ctx context.Context, object *Object) (*Object, int64, error) {
	if object == nil {
		logrus.WithContext(ctx).WithFields(logrus.Fields{}).Warnf("更新%s，为空", this.GetName(ctx))
		return object, 0, nil
	}
	where := this.GetDb(ctx, nil)
	where = where.Model(&object)
	result := where.Select("*").Updates(&object)
	count := result.RowsAffected
	err := result.Error
	if err != nil {
		logrus.WithContext(ctx).WithFields(logrus.Fields{"err": err}).Errorf("更新%s，异常", this.GetName(ctx))
		return object, count, errors.Errorf("更新%s，异常: %+v", this.GetName(ctx), err)
	}
	logrus.WithContext(ctx).WithFields(logrus.Fields{}).Infof("更新%s，完成", this.GetName(ctx))
	return object, count, nil
}
func (this *GormService[Object, Inquiry]) Select(ctx context.Context, inquiry Inquiry) ([]*Object, int64, error) {
	where := this.GetDb(ctx, nil)
	where = where.Model(&inquiry)
	where = this.Where(ctx, where, inquiry)

	var count int64
	err := where.Count(&count).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		logrus.WithContext(ctx).WithFields(logrus.Fields{}).Warnf("查询%s，不存在", this.GetName(ctx))
		return nil, count, nil
	}
	if err != nil {
		logrus.WithContext(ctx).WithFields(logrus.Fields{"err": err}).Warnf("查询%s，异常", this.GetName(ctx))
		return nil, count, errors.Errorf("查询%s，异常: %+v", this.GetName(ctx), err)
	}

	offset := inquiry.GetOffset()
	if offset > 0 {
		where = where.Offset(offset)
	}
	limit := inquiry.GetLimit()
	if limit > 0 {
		where = where.Limit(limit)
	}

	var object []*Object
	err = where.Find(&object).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		logrus.WithContext(ctx).WithFields(logrus.Fields{}).Warnf("查询%s，不存在", this.GetName(ctx))
		return object, count, nil
	}
	if err != nil {
		logrus.WithContext(ctx).WithFields(logrus.Fields{"err": err}).Errorf("查询%s，异常", this.GetName(ctx))
		return object, count, errors.Errorf("查询%s, 异常: %+v", this.GetName(ctx), err)
	}
	logrus.WithContext(ctx).WithFields(logrus.Fields{"len": len(object)}).Infof("查询%s，完成", this.GetName(ctx))
	return object, count, nil
}
func (this *GormService[Object, Inquiry]) SelectOne(ctx context.Context, inquiry Inquiry) (*Object, error) {
	list, _, err := this.Select(ctx, inquiry)
	if err != nil {
		return nil, err
	}
	if len(list) == 0 {
		return nil, nil
	}
	return list[0], nil
}
