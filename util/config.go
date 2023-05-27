package util

import (
	"context"
	"fmt"
	"github.com/sirupsen/logrus"
	"sync"
	"time"
)

type ConfigHandler interface {
	GetPath(ctx context.Context) string
	GetConfig(ctx context.Context) string
	ParseConfig(ctx context.Context, text string) error
}

func NewConfigService(handler ConfigHandler) *ConfigService {
	var service ConfigService
	service.handler = handler
	service.lock = &sync.Mutex{}
	return &service
}

type ConfigService struct {
	handler ConfigHandler
	lock    *sync.Mutex
	pool    *SingleGoPool
	text    string
}

func (this *ConfigService) Start(ctx context.Context) error {
	this.lock.Lock()
	defer this.lock.Unlock()

	if this.pool != nil && !this.pool.IsClose(ctx) {
		logrus.WithContext(ctx).WithFields(logrus.Fields{}).Info("ConfigService，已启动")
		return nil
	}

	var err error
	this.pool, err = NewDaemonSingleGoPool(ctx, fmt.Sprintf("ConfigService-%s", this.handler.GetPath(ctx)), time.Minute, this.flushConfig)
	if err != nil {
		return err
	}
	logrus.WithContext(ctx).WithFields(logrus.Fields{}).Info("ConfigService，启动")
	return this.loadConfig(ctx)
}
func (this *ConfigService) flushConfig(ctx context.Context, pool *SingleGoPool) {
	defer Defer(func(err interface{}, stack string) {
		if err != nil {
			logrus.WithContext(ctx).WithFields(logrus.Fields{"err": err, "stack": stack}).Error("ConfigService，异常")
		}
	})

	for {
		ctx := ResetLogId(ctx)
		this.LoadConfig(ctx)
		Sleep(ctx, time.Minute)
		if CtxDone(ctx) {
			return
		}
	}
}
func (this *ConfigService) SaveConfig(ctx context.Context) error {
	this.lock.Lock()
	defer this.lock.Unlock()

	if this.text == "" {
		this.text = this.handler.GetConfig(ctx)
	}
	logrus.WithContext(ctx).WithFields(logrus.Fields{}).Info("ConfigService，保存")
	return WriteString2File(ctx, this.text, this.handler.GetPath(ctx))
}
func (this *ConfigService) LoadConfig(ctx context.Context) error {
	this.lock.Lock()
	defer this.lock.Unlock()

	return this.loadConfig(ctx)
}
func (this *ConfigService) loadConfig(ctx context.Context) error {
	text, err := ReadFile2String(ctx, this.handler.GetPath(ctx), "")
	if err != nil {
		return err
	}
	if text == "" {
		text = this.handler.GetConfig(ctx)
		err = WriteString2File(ctx, this.text, this.handler.GetPath(ctx))
		if err != nil {
			return err
		}
	}
	logrus.WithContext(ctx).WithFields(logrus.Fields{}).Info("ConfigService，加载")
	if text == this.text {
		return nil
	}
	err = this.handler.ParseConfig(ctx, text)
	if err != nil {
		return err
	}
	this.text = text
	return nil
}
