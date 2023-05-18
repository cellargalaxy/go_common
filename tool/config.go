package tool

import (
	"context"
	"fmt"
	"github.com/cellargalaxy/go_common/util"
	"github.com/sirupsen/logrus"
	"sync"
	"time"
)

type ConfigHandler interface {
	GetConfig(ctx context.Context) string
	ParseConfig(ctx context.Context, text string)
}

func NewConfigService(filePath string, handler ConfigHandler) *ConfigService {
	var service ConfigService
	service.FilePath = filePath
	service.handler = handler
	service.lock = &sync.Mutex{}
	return &service
}

type ConfigService struct {
	FilePath string `json:"file_path"`

	handler ConfigHandler
	lock    *sync.Mutex
	pool    *util.SingleGoPool
	text    string
}

func (this *ConfigService) Start(ctx context.Context) error {
	this.lock.Lock()
	defer this.lock.Unlock()

	if this.pool != nil && !this.pool.IsClose(ctx) {
		return nil
	}

	var err error
	this.pool, err = util.NewDaemonSingleGoPool(ctx, fmt.Sprintf(""), time.Minute, this.flushConfig)
	if err != nil {
		return err
	}
	return nil
}
func (this *ConfigService) flushConfig(ctx context.Context, pool *util.SingleGoPool) {
	defer util.Defer(func(err interface{}, stack string) {
		if err != nil {
			logrus.WithContext(ctx).WithFields(logrus.Fields{"err": err, "stack": stack}).Warn("ConfigService，异常")
		}
	})

	for {
		ctx := util.ResetLogId(ctx)
		this.LoadConfig(ctx)
		util.Sleep(ctx, time.Minute)
		if util.CtxDone(ctx) {
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
	return util.WriteString2File(ctx, this.text, this.FilePath)
}
func (this *ConfigService) LoadConfig(ctx context.Context) error {
	this.lock.Lock()
	defer this.lock.Unlock()

	text, err := util.ReadFile2String(ctx, this.FilePath, "")
	if err != nil {
		return err
	}
	if text == "" {
		text = this.handler.GetConfig(ctx)
		err = util.WriteString2File(ctx, this.text, this.FilePath)
		if err != nil {
			return err
		}
	}
	if text == this.text {
		return nil
	}
	this.text = text
	this.handler.ParseConfig(ctx, text)
	return nil
}
