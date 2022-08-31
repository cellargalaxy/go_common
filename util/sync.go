package util

import (
	"context"
	"fmt"
	"github.com/panjf2000/ants/v2"
	"github.com/sirupsen/logrus"
	"sync"
	"time"
)

func Cancel(cancels ...func()) {
	for i := range cancels {
		if cancels[i] == nil {
			continue
		}
		cancels[i]()
	}
}

func ReleasePool(pools ...*ants.Pool) {
	for i := range pools {
		if pools[i] == nil {
			continue
		}
		pools[i].Release()
	}
}

func NewOnceSingleGoPool(ctx context.Context, name string, task func(ctx context.Context, pool *SingleGoPool)) (*SingleGoPool, error) {
	pool, err := NewSingleGoPool(ctx, "")
	if err != nil {
		return nil, err
	}
	err = pool.AddOnceTask(ctx, name, task)
	if err != nil {
		pool.Release(ctx)
		return nil, err
	}
	return pool, nil
}

func NewForeverSingleGoPool(ctx context.Context, name string, sleep time.Duration, task func(ctx context.Context, pool *SingleGoPool)) (*SingleGoPool, error) {
	pool, err := NewSingleGoPool(ctx, "")
	if err != nil {
		return nil, err
	}
	err = pool.AddForeverTask(ctx, name, sleep, task)
	if err != nil {
		pool.Release(ctx)
		return nil, err
	}
	return pool, nil
}

func NewSingleGoPool(ctx context.Context, name string) (*SingleGoPool, error) {
	pool, err := ants.NewPool(1, ants.WithMaxBlockingTasks(1))
	if pool == nil {
		logrus.WithContext(ctx).WithFields(logrus.Fields{"name": name}).Error("创建单协程池，为空")
		return nil, fmt.Errorf("创建单协程池，为空")
	}
	if err != nil {
		ReleasePool(pool)
		logrus.WithContext(ctx).WithFields(logrus.Fields{"name": name, "err": err}).Error("创建单协程池，异常")
		return nil, fmt.Errorf("创建单协程池，异常: %+v", err)
	}

	return &SingleGoPool{poolName: name, pool: pool}, nil
}

type SingleGoPool struct {
	poolName string
	pool     *ants.Pool

	lock     sync.Mutex
	taskName string
	cancel   func()
}

func (this *SingleGoPool) AddForeverTask(ctx context.Context, name string, sleep time.Duration, task func(ctx context.Context, pool *SingleGoPool)) error {
	this.lock.Lock()
	defer this.lock.Unlock()

	if this.taskName == name {
		logrus.WithContext(ctx).WithFields(logrus.Fields{"name": name}).Info("单协程池，永久任务已添加")
		return nil
	}

	this.Cancel(ctx)
	this.taskName = name
	ctx, this.cancel = context.WithCancel(ctx)

	var err error
	err = this.addForeverTask(ctx, name, sleep, task)
	if err != nil {
		this.Cancel(ctx)
		return err
	}

	return nil
}
func (this *SingleGoPool) addForeverTask(ctx context.Context, name string, sleep time.Duration, task func(ctx context.Context, pool *SingleGoPool)) error {
	submit := func() {
		defer Defer(func(err interface{}, stack string) {
			if err == nil {
				logrus.WithContext(ctx).WithFields(logrus.Fields{"name": this.GetName(ctx)}).Info("单协程池，退出")
			} else {
				logrus.WithContext(ctx).WithFields(logrus.Fields{"name": this.GetName(ctx), "err": err, "stack": stack}).Error("单协程池，退出")
			}
			this.taskName = ""

			go func() {
				Sleep(ctx, sleep)
				if CtxDone(ctx) {
					logrus.WithContext(ctx).WithFields(logrus.Fields{"name": this.GetName(ctx)}).Info("单协程池，已取消")
					return
				}
				if this.pool.IsClosed() {
					logrus.WithContext(ctx).WithFields(logrus.Fields{"name": this.GetName(ctx)}).Info("单协程池，已释放")
					return
				}
				this.addForeverTask(ctx, name, sleep, task)
			}()
		})

		task(ctx, this)
	}

	err := this.pool.Submit(submit)
	if err != nil {
		logrus.WithContext(ctx).WithFields(logrus.Fields{"name": this.GetName(ctx), "err": err}).Error("单协程池，添加永久任务异常")
		this.Cancel(ctx)
		return fmt.Errorf("单协程池，添加永久任务异常: %+v", err)
	}

	return nil
}
func (this *SingleGoPool) AddOnceTask(ctx context.Context, name string, task func(ctx context.Context, pool *SingleGoPool)) error {
	this.lock.Lock()
	defer this.lock.Unlock()

	if this.taskName == name {
		logrus.WithContext(ctx).WithFields(logrus.Fields{"name": name}).Info("单协程池，单次任务已添加")
		return nil
	}

	this.Cancel(ctx)
	this.taskName = name
	ctx, this.cancel = context.WithCancel(ctx)

	var err error
	err = this.addOnceTask(ctx, name, task)
	if err != nil {
		this.Cancel(ctx)
		return err
	}

	return nil
}
func (this *SingleGoPool) addOnceTask(ctx context.Context, name string, task func(ctx context.Context, pool *SingleGoPool)) error {
	this.lock.Lock()
	defer this.lock.Unlock()

	if this.taskName == name {
		logrus.WithContext(ctx).WithFields(logrus.Fields{"name": name}).Info("单协程池，单次任务已添加")
		return nil
	}

	this.Cancel(ctx)
	this.taskName = name
	ctx, this.cancel = context.WithCancel(ctx)

	submit := func() {
		defer Defer(func(err interface{}, stack string) {
			if err == nil {
				logrus.WithContext(ctx).WithFields(logrus.Fields{"name": this.GetName(ctx)}).Info("单协程池，退出")
			} else {
				logrus.WithContext(ctx).WithFields(logrus.Fields{"name": this.GetName(ctx), "err": err, "stack": stack}).Error("单协程池，退出")
			}
			this.taskName = ""
		})

		task(ctx, this)
	}

	err := this.pool.Submit(submit)
	if err != nil {
		logrus.WithContext(ctx).WithFields(logrus.Fields{"name": this.GetName(ctx), "err": err}).Error("单协程池，添加单次任务异常")
		this.Cancel(ctx)
		return fmt.Errorf("单协程池，添加单次任务异常: %+v", err)
	}

	return nil
}
func (this *SingleGoPool) Cancel(ctx context.Context) {
	logrus.WithContext(ctx).WithFields(logrus.Fields{"name": this.GetName(ctx)}).Info("单协程池，取消")
	Cancel(this.cancel)
	this.taskName = ""
}
func (this *SingleGoPool) Release(ctx context.Context) {
	logrus.WithContext(ctx).WithFields(logrus.Fields{"name": this.GetName(ctx)}).Info("单协程池，释放")
	this.Cancel(ctx)
	ReleasePool(this.pool)
}
func (this *SingleGoPool) GetName(ctx context.Context) string {
	if this.poolName != "" && this.taskName != "" {
		return fmt.Sprintf("%s_%s", this.poolName, this.taskName)
	}
	if this.poolName != "" {
		return this.poolName
	}
	if this.taskName != "" {
		return this.taskName
	}
	return "SingleGoPool"
}
