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

type SingleGoPool struct {
	poolName string
	pool     *ants.Pool

	lock     sync.Mutex
	taskName string
	cancel   func()
}

func (this *SingleGoPool) AddTask(ctx context.Context, name string, sleep time.Duration, task func(ctx context.Context, pool *SingleGoPool)) error {
	this.lock.Lock()
	defer this.lock.Unlock()

	if this.taskName == name {
		logrus.WithContext(ctx).WithFields(logrus.Fields{"name": name}).Info("单协程池，任务已添加")
		return nil
	}

	this.Cancel(ctx)
	this.taskName = name

	return nil
}
func (this *SingleGoPool) addTask(ctx context.Context, sleep time.Duration, task func(ctx context.Context, pool *SingleGoPool)) error {
	ctx, this.cancel = context.WithCancel(ctx)

	var submit func()
	submit = func() {
		defer func() {
			logrus.WithContext(ctx).WithFields(logrus.Fields{"name": this.GetName(ctx)}).Info("单协程池，退出")
			go func() {
				Sleep(ctx, sleep)
				if CtxDone(ctx) {
					logrus.WithContext(ctx).WithFields(logrus.Fields{"name": this.GetName(ctx)}).Info("单协程池，结束")
					return
				}
				if this.pool.IsClosed() {
					logrus.WithContext(ctx).WithFields(logrus.Fields{"name": this.GetName(ctx)}).Info("单协程池，关闭")
					return
				}

				err := this.pool.Submit(submit)
				if err != nil {
					logrus.WithContext(ctx).WithFields(logrus.Fields{"name": this.GetName(ctx), "err": err}).Error("单协程池，添加任务异常")
				}
			}()
		}()

		task(ctx, this)
	}

	err := this.pool.Submit(submit)
	if err != nil {
		logrus.WithContext(ctx).WithFields(logrus.Fields{"name": this.GetName(ctx), "err": err}).Error("单协程池，添加任务异常")
		this.Cancel(ctx)
		return fmt.Errorf("单协程池，添加任务异常: %+v", err)
	}

	return nil
}
func (this *SingleGoPool) Cancel(ctx context.Context) {
	logrus.WithContext(ctx).WithFields(logrus.Fields{"name": this.GetName(ctx)}).Info("单协程池，取消任务")
	Cancel(this.cancel)
}
func (this *SingleGoPool) Release(ctx context.Context) {
	logrus.WithContext(ctx).WithFields(logrus.Fields{"name": this.GetName(ctx)}).Info("单协程池，释放协程池")
	this.Cancel(ctx)
	ReleasePool(this.pool)
}
func (this *SingleGoPool) GetName(ctx context.Context) string {
	return fmt.Sprintf("%s_%s", this.poolName, this.taskName)
}

func NewForeverSingleGoPool(ctx context.Context, name string, sleep time.Duration, task func(ctx context.Context, cancel func())) (*ants.Pool, error) {
	ctx, cancel := context.WithCancel(ctx)
	pool, err := ants.NewPool(1, ants.WithMaxBlockingTasks(1))
	if pool == nil {
		cancel()
		logrus.WithContext(ctx).WithFields(logrus.Fields{"name": name}).Error("创建常驻单协程池，为空")
		return nil, fmt.Errorf("创建常驻单协程池，为空")
	}
	if err != nil {
		cancel()
		ReleasePool(pool)
		logrus.WithContext(ctx).WithFields(logrus.Fields{"name": name, "err": err}).Error("创建常驻单协程池，异常")
		return nil, fmt.Errorf("创建常驻单协程池，异常: %+v", err)
	}

	var submit func()
	submit = func() {
		defer func() {
			logrus.WithContext(ctx).WithFields(logrus.Fields{"name": name}).Info("常驻单协程池，退出")
			go func() {
				Sleep(ctx, sleep)
				if CtxDone(ctx) {
					ReleasePool(pool)
					logrus.WithContext(ctx).WithFields(logrus.Fields{"name": name}).Info("常驻单协程池，结束")
					return
				}
				if pool.IsClosed() {
					cancel()
					logrus.WithContext(ctx).WithFields(logrus.Fields{"name": name}).Info("常驻单协程池，关闭")
					return
				}

				err := pool.Submit(submit)
				if err != nil {
					logrus.WithContext(ctx).WithFields(logrus.Fields{"name": name, "err": err}).Error("创建常驻单协程池，添加任务异常")
				}
			}()
		}()

		task(ctx, cancel)
	}

	err = pool.Submit(submit)
	if err != nil {
		cancel()
		ReleasePool(pool)
		logrus.WithContext(ctx).WithFields(logrus.Fields{"name": name, "err": err}).Error("创建常驻单协程池，添加任务异常")
		return nil, fmt.Errorf("创建常驻单协程池，添加任务异常: %+v", err)
	}
	return pool, nil
}
