package util

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

type ConfigHandler interface {
	GetPath(ctx context.Context) string
	GetDefault(ctx context.Context) string
	Parse(ctx context.Context, text string) error
}

func NewConfigService(handler ConfigHandler) *ConfigService {
	var service ConfigService
	service.handler = handler
	service.lock = &sync.Mutex{}
	return &service
}

type ConfigService struct {
	handler ConfigHandler
	lock    *sync.Mutex   // 如果不使用指针会有问题吗
	pool    *SingleGoPool // 如果不使用指针会有问题吗

	text string
}

func (this *ConfigService) Start(ctx context.Context) error {
	this.lock.Lock()
	defer this.lock.Unlock()

	if this.pool != nil && !this.pool.IsClose(ctx) {
		logrus.WithContext(ctx).WithFields(logrus.Fields{}).Info("ConfigService，已启动")
		return nil
	}

	var err error
	configPath := this.handler.GetPath(ctx)
	this.pool, err = NewDaemonSingleGoPool(ctx, fmt.Sprintf("ConfigService-%s", configPath), time.Minute, this.flushConfig)
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
		ccc := ReSetLogId(ctx)
		this.LoadConfig(ccc)
		Sleep(ccc, time.Minute)
		if CtxDone(ccc) {
			return
		}
	}
}
func (this *ConfigService) LoadConfig(ctx context.Context) error {
	this.lock.Lock()
	defer this.lock.Unlock()

	return this.loadConfig(ctx)
}
func (this *ConfigService) loadConfig(ctx context.Context) error {
	configPath := this.handler.GetPath(ctx)
	logrus.WithContext(ctx).WithFields(logrus.Fields{"path": configPath}).Info("ConfigService，加载")
	text, err := ReadFile2Str(ctx, configPath, "")
	if err != nil {
		return err
	}
	if text == "" {
		text = this.handler.GetDefault(ctx)
		err = WriteStr2File(ctx, text, configPath)
		if err != nil {
			return err
		}
	}
	if text == this.text {
		return nil
	}
	logrus.WithContext(ctx).WithFields(logrus.Fields{}).Info("ConfigService，解析")
	err = this.handler.Parse(ctx, text)
	if err != nil {
		return err
	}
	this.text = text
	return nil
}
