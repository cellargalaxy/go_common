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
	service.textLock = &sync.RWMutex{}
	return &service
}

type ConfigService struct {
	handler ConfigHandler
	lock    *sync.Mutex
	pool    *SingleGoPool

	// text 由守护协程的 loadConfig 与调用方的 SetConfig/GetConfig 并发访问，
	// 必须加锁保护(go test -race 可复现裸读写的数据竞争)。
	// 这里专门用一把独立的锁而不复用 this.lock：this.lock 在 loadConfig 期间
	// 会跨越 handler.ParseConfig 回调，若该回调内部再调 GetConfig，
	// 复用同一把不可重入的锁会直接死锁。
	textLock *sync.RWMutex
	text     string
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

	return this.saveConfig(ctx)
}
func (this *ConfigService) saveConfig(ctx context.Context) error {
	text := this.loadText()
	if text == "" {
		text = this.handler.GetConfig(ctx)
		this.storeText(text)
	}
	logrus.WithContext(ctx).WithFields(logrus.Fields{}).Info("ConfigService，保存")
	return WriteString2File(ctx, text, this.handler.GetPath(ctx))
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
		//落盘默认配置，此处不能用this.text，其此刻仍为空会写出空文件
		err = WriteString2File(ctx, text, this.handler.GetPath(ctx))
		if err != nil {
			return err
		}
	}
	logrus.WithContext(ctx).WithFields(logrus.Fields{}).Info("ConfigService，加载")
	if text == this.loadText() {
		return nil
	}
	err = this.handler.ParseConfig(ctx, text)
	if err != nil {
		return err
	}
	this.storeText(text)
	return nil
}

// loadText/storeText 是 text 字段的唯一读写入口，统一走 textLock
func (this *ConfigService) loadText() string {
	this.textLock.RLock()
	defer this.textLock.RUnlock()

	return this.text
}
func (this *ConfigService) storeText(text string) {
	this.textLock.Lock()
	defer this.textLock.Unlock()

	this.text = text
}
func (this *ConfigService) GetConfig(ctx context.Context) string {
	return this.loadText()
}
func (this *ConfigService) SetConfig(ctx context.Context, text string) {
	this.storeText(text)
}
