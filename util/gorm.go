package util

import (
	"context"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

const (
	LogSqlLen   = 512
	DbBatchSize = 1000
)

func NewDefaultGormLog() GormLog {
	return NewGormLog([]error{gorm.ErrRecordNotFound}, LogSqlLen, false, true, true, false, true)
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

type TransactionHandler interface {
	Transaction(ctx context.Context, tx *gorm.DB) error
}

func Transaction(ctx context.Context, db *gorm.DB, handlers ...TransactionHandler) error {
	err := db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for i := range handlers {
			err := handlers[i].Transaction(ctx, tx)
			if err != nil {
				return err
			}
		}
		return nil
	})
	return err
}

func NewInsertHandler[Object any](name string, object ...*Object) *InsertHandler[Object] {
	handler := new(InsertHandler[Object])
	handler.name = name
	handler.Object = object
	return handler
}

type InsertHandler[Object any] struct {
	name   string
	Object []*Object
	Count  int64
}

func (this *InsertHandler[Object]) Transaction(ctx context.Context, tx *gorm.DB) error {
	if len(this.Object) == 0 {
		logrus.WithContext(ctx).WithFields(logrus.Fields{}).Warnf("插入%s，为空", this.name)
		return nil
	}
	result := tx.CreateInBatches(this.Object, DbBatchSize)
	this.Count = result.RowsAffected
	err := result.Error
	if err != nil {
		logrus.WithContext(ctx).WithFields(logrus.Fields{"err": err}).Errorf("插入%s，异常", this.name)
		return errors.Errorf("插入%s，异常: %+v", this.name, err)
	}
	logrus.WithContext(ctx).WithFields(logrus.Fields{"count": this.Count}).Infof("插入%s，完成", this.name)
	return nil
}

func NewUpdateHandler[Object any](name string, object *Object) *UpdateHandler[Object] {
	handler := new(UpdateHandler[Object])
	handler.name = name
	handler.Object = object
	return handler
}

type UpdateHandler[Object any] struct {
	name   string
	Object *Object
	Count  int64
}

func (this *UpdateHandler[Object]) Transaction(ctx context.Context, tx *gorm.DB) error {
	if this.Object == nil {
		logrus.WithContext(ctx).WithFields(logrus.Fields{}).Warnf("更新%s，为空", this.name)
		return nil
	}
	tx = tx.Model(this.Object)
	result := tx.Select("*").Updates(this.Object)
	this.Count = result.RowsAffected
	err := result.Error
	if err != nil {
		logrus.WithContext(ctx).WithFields(logrus.Fields{"err": err}).Errorf("更新%s，异常", this.name)
		return errors.Errorf("更新%s，异常: %+v", this.name, err)
	}
	logrus.WithContext(ctx).WithFields(logrus.Fields{"count": this.Count}).Infof("更新%s，完成", this.name)
	return nil
}

type InquiryHandler[Inquiry any] interface {
	Where(ctx context.Context, tx *gorm.DB, inquiry Inquiry) *gorm.DB
	Order(ctx context.Context, tx *gorm.DB, inquiry Inquiry) *gorm.DB
	Limit(ctx context.Context, tx *gorm.DB, inquiry Inquiry) *gorm.DB
}

func NewDeleteHandler[Object any, Inquiry any](name string, inquiry Inquiry, inquiryHandler InquiryHandler[Inquiry]) *DeleteHandler[Object, Inquiry] {
	handler := new(DeleteHandler[Object, Inquiry])
	handler.name = name
	handler.inquiry = inquiry
	handler.InquiryHandler = inquiryHandler
	return handler
}

type DeleteHandler[Object any, Inquiry any] struct {
	InquiryHandler[Inquiry]
	name    string
	inquiry Inquiry
	Count   int64
}

func (this *DeleteHandler[Object, Inquiry]) Transaction(ctx context.Context, tx *gorm.DB) error {
	tx = this.Where(ctx, tx, this.inquiry)
	tx = this.Order(ctx, tx, this.inquiry)
	tx = this.Limit(ctx, tx, this.inquiry)
	result := tx.Delete(new(Object))
	this.Count = result.RowsAffected
	err := result.Error
	if err != nil {
		logrus.WithContext(ctx).WithFields(logrus.Fields{"err": err}).Errorf("删除%s，异常", this.name)
		return errors.Errorf("删除%s，异常: %+v", this.name, err)
	}
	logrus.WithContext(ctx).WithFields(logrus.Fields{"count": this.Count}).Infof("删除%s，完成", this.name)
	return nil
}

func NewSelectHandler[Object any, Inquiry any](name string, inquiry Inquiry, inquiryHandler InquiryHandler[Inquiry]) *SelectHandler[Object, Inquiry] {
	handler := new(SelectHandler[Object, Inquiry])
	handler.name = name
	handler.inquiry = inquiry
	handler.InquiryHandler = inquiryHandler
	handler.Object = make([]*Object, 0)
	return handler
}

type SelectHandler[Object any, Inquiry any] struct {
	InquiryHandler[Inquiry]
	name    string
	inquiry Inquiry
	Object  []*Object
	Count   int64
}

func (this *SelectHandler[Object, Inquiry]) Transaction(ctx context.Context, tx *gorm.DB) error {
	tx = tx.Model(new(Object))
	tx = this.Where(ctx, tx, this.inquiry)

	err := tx.Count(&this.Count).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		logrus.WithContext(ctx).WithFields(logrus.Fields{}).Warnf("查询%s，不存在", this.name)
		return nil
	}
	if err != nil {
		logrus.WithContext(ctx).WithFields(logrus.Fields{"err": err}).Warnf("查询%s，异常", this.name)
		return errors.Errorf("查询%s，异常: %+v", this.name, err)
	}

	tx = this.Order(ctx, tx, this.inquiry)
	tx = this.Limit(ctx, tx, this.inquiry)
	err = tx.Find(&this.Object).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		logrus.WithContext(ctx).WithFields(logrus.Fields{}).Warnf("查询%s，不存在", this.name)
		return nil
	}
	if err != nil {
		logrus.WithContext(ctx).WithFields(logrus.Fields{"err": err}).Errorf("查询%s，异常", this.name)
		return errors.Errorf("查询%s, 异常: %+v", this.name, err)
	}
	logrus.WithContext(ctx).WithFields(logrus.Fields{"len": len(this.Object)}).Infof("查询%s，完成", this.name)
	return nil
}
func (this *SelectHandler[Object, Inquiry]) GetOne() *Object {
	if len(this.Object) == 0 {
		return nil
	}
	return this.Object[0]
}
