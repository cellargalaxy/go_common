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

func Release(pools ...*ants.Pool) {
	for i := range pools {
		if pools[i] == nil {
			continue
		}
		pools[i].Release()
	}
}

func CancelPool(ctx context.Context, pools ...*SingleGoPool) {
	for i := range pools {
		if pools[i] == nil {
			continue
		}
		pools[i].Cancel(ctx)
	}
}

func ReleasePool(ctx context.Context, pools ...*SingleGoPool) {
	for i := range pools {
		if pools[i] == nil {
			continue
		}
		pools[i].Release(ctx)
	}
}

func NewOnceSingleGoPool(ctx context.Context, name string, task func(cancelCtx context.Context, pool *SingleGoPool)) (*SingleGoPool, error) {
	pool, err := NewSingleGoPool(ctx, "")
	if err != nil {
		return nil, err
	}
	err = pool.AddOnceTask(ctx, name, task)
	if err != nil {
		ReleasePool(ctx, pool)
		return nil, err
	}
	return pool, nil
}

func NewDaemonSingleGoPool(ctx context.Context, name string, sleep time.Duration, task func(cancelCtx context.Context, pool *SingleGoPool)) (*SingleGoPool, error) {
	pool, err := NewSingleGoPool(ctx, "")
	if err != nil {
		return nil, err
	}
	err = pool.AddDaemonTask(ctx, name, sleep, task)
	if err != nil {
		ReleasePool(ctx, pool)
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
		Release(pool)
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

func (this *SingleGoPool) AddDaemonTask(ctx context.Context, name string, sleep time.Duration, task func(cancelCtx context.Context, pool *SingleGoPool)) error {
	this.lock.Lock()
	defer this.lock.Unlock()

	if this.taskName == name {
		logrus.WithContext(ctx).WithFields(logrus.Fields{"name": name}).Info("单协程池，守护任务已添加")
		return nil
	}

	CancelPool(ctx, this)
	this.taskName = name
	ctx, this.cancel = context.WithCancel(ctx)

	var err error
	err = this.addDaemonTask(ctx, name, sleep, task)
	if err != nil {
		CancelPool(ctx, this)
		return err
	}

	return nil
}
func (this *SingleGoPool) addDaemonTask(ctx context.Context, name string, sleep time.Duration, task func(cancelCtx context.Context, pool *SingleGoPool)) error {
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
				this.addDaemonTask(ctx, name, sleep, task)
			}()
		})

		task(ctx, this)
	}

	err := this.pool.Submit(submit)
	if err != nil {
		CancelPool(ctx, this)
		logrus.WithContext(ctx).WithFields(logrus.Fields{"name": this.GetName(ctx), "err": err}).Error("单协程池，添加守护任务异常")
		return fmt.Errorf("单协程池，添加守护任务异常: %+v", err)
	}

	return nil
}
func (this *SingleGoPool) AddOnceTask(ctx context.Context, name string, task func(cancelCtx context.Context, pool *SingleGoPool)) error {
	this.lock.Lock()
	defer this.lock.Unlock()

	if this.taskName == name {
		logrus.WithContext(ctx).WithFields(logrus.Fields{"name": name}).Info("单协程池，单次任务已添加")
		return nil
	}

	CancelPool(ctx, this)
	this.taskName = name
	ctx, this.cancel = context.WithCancel(ctx)

	var err error
	err = this.addOnceTask(ctx, task)
	if err != nil {
		CancelPool(ctx, this)
		return err
	}

	return nil
}
func (this *SingleGoPool) addOnceTask(ctx context.Context, task func(cancelCtx context.Context, pool *SingleGoPool)) error {
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
		CancelPool(ctx, this)
		logrus.WithContext(ctx).WithFields(logrus.Fields{"name": this.GetName(ctx), "err": err}).Error("单协程池，添加单次任务异常")
		return fmt.Errorf("单协程池，添加单次任务异常: %+v", err)
	}

	return nil
}
func (this *SingleGoPool) Doing(ctx context.Context) bool {
	return this.taskName != ""
}
func (this *SingleGoPool) IsClosed(ctx context.Context) bool {
	isClosed := this.pool.IsClosed()
	logrus.WithContext(ctx).WithFields(logrus.Fields{"name": this.GetName(ctx), "isClosed": isClosed}).Info("单协程池")
	return isClosed
}
func (this *SingleGoPool) Cancel(ctx context.Context) {
	logrus.WithContext(ctx).WithFields(logrus.Fields{"name": this.GetName(ctx)}).Info("单协程池，取消")
	Cancel(this.cancel)
	this.taskName = ""
}
func (this *SingleGoPool) Release(ctx context.Context) {
	logrus.WithContext(ctx).WithFields(logrus.Fields{"name": this.GetName(ctx)}).Info("单协程池，释放")
	Cancel(this.cancel)
	Release(this.pool)
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
