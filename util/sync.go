package util

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/panjf2000/ants/v2"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

func CancelPool(ctx context.Context, pools ...*SingleGoPool) {
	for i := range pools {
		if pools[i] == nil {
			continue
		}
		pools[i].Cancel(ctx)
	}
}
func ClosePool(ctx context.Context, pools ...*SingleGoPool) {
	for i := range pools {
		if pools[i] == nil {
			continue
		}
		pools[i].Close(ctx)
	}
}

func NewOnceSingleGoPool(ctx context.Context, name string, task func(cancelCtx context.Context, pool *SingleGoPool)) (*SingleGoPool, error) {
	pool, err := NewSingleGoPool(ctx, "")
	if err != nil {
		return nil, err
	}
	err = pool.AddOnceTask(ctx, name, task)
	if err != nil {
		ClosePool(ctx, pool)
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
		ClosePool(ctx, pool)
		return nil, err
	}
	return pool, nil
}
func NewSingleGoPool(ctx context.Context, name string) (*SingleGoPool, error) {
	pool, err := ants.NewPool(1, ants.WithMaxBlockingTasks(1))
	if pool == nil {
		logrus.WithContext(ctx).WithFields(logrus.Fields{"name": name}).Error("创建单协程池，为空")
		return nil, errors.Errorf("创建单协程池，为空")
	}
	if err != nil {
		pool.Release()
		logrus.WithContext(ctx).WithFields(logrus.Fields{"name": name, "err": err}).Error("创建单协程池，异常")
		return nil, errors.Errorf("创建单协程池，异常: %+v", err)
	}

	return &SingleGoPool{poolName: name, pool: pool, lock: &sync.RWMutex{}}, nil
}

type SingleGoPool struct {
	poolName  string
	pool      *ants.Pool    //如果不使用指针会有问题吗
	lock      *sync.RWMutex //如果不使用指针会有问题吗
	taskName  string
	ctxCancel func()
}

func (this *SingleGoPool) AddDaemonTask(ctx context.Context, name string, sleep time.Duration, task func(cancelCtx context.Context, pool *SingleGoPool)) error {
	this.lock.Lock()
	defer this.lock.Unlock()

	var err error
	err = this.addDaemonTask(ctx, name, sleep, task)
	if err != nil {
		return err
	}
	return nil
}
func (this *SingleGoPool) addDaemonTask(ctx context.Context, name string, sleep time.Duration, task func(cancelCtx context.Context, pool *SingleGoPool)) error {
	submit := func() {
		defer Defer(func(panic any, stack string) {
			if panic == nil {
				logrus.WithContext(ctx).WithFields(logrus.Fields{"name": this.getName()}).Info("单协程池，退出")
			} else {
				logrus.WithContext(ctx).WithFields(logrus.Fields{"name": this.getName(), "panic": panic, "stack": stack}).Error("单协程池，退出")
			}

			go func() {
				this.lock.Lock()
				defer this.lock.Unlock()

				if this.taskName == name {
					this.taskName = ""
				}
				Sleep(ctx, sleep)
				this.addDaemonTask(ctx, name, sleep, task)
			}()
		})

		task(ctx, this)
	}

	if name == "" {
		name = GenStrId()
	}
	if this.taskName == name {
		logrus.WithContext(ctx).WithFields(logrus.Fields{"name": name}).Warn("单协程池，守护任务已添加")
		return nil
	}
	this.cancel(ctx)
	ctx, this.ctxCancel = context.WithCancel(ctx)

	if CtxDone(ctx) {
		this.cancel(ctx)
		logrus.WithContext(ctx).WithFields(logrus.Fields{"name": this.getName()}).Info("单协程池，已取消")
		return nil
	}
	if this.isClose(ctx) {
		this.cancel(ctx)
		logrus.WithContext(ctx).WithFields(logrus.Fields{"name": this.getName()}).Info("单协程池，已关闭")
		return nil
	}
	err := this.pool.Submit(submit)
	if err != nil {
		this.cancel(ctx)
		logrus.WithContext(ctx).WithFields(logrus.Fields{"name": this.getName(), "err": err}).Error("单协程池，添加守护任务异常")
		return errors.Errorf("单协程池，添加守护任务异常: %+v", err)
	}
	this.taskName = name

	return nil
}
func (this *SingleGoPool) AddOnceTask(ctx context.Context, name string, task func(cancelCtx context.Context, pool *SingleGoPool)) error {
	this.lock.Lock()
	defer this.lock.Unlock()

	err := this.addOnceTask(ctx, name, task)
	if err != nil {
		return err
	}
	return nil
}
func (this *SingleGoPool) addOnceTask(ctx context.Context, name string, task func(cancelCtx context.Context, pool *SingleGoPool)) error {
	submit := func() {
		defer Defer(func(panic any, stack string) {
			if panic == nil {
				logrus.WithContext(ctx).WithFields(logrus.Fields{"name": this.getName()}).Info("单协程池，退出")
			} else {
				logrus.WithContext(ctx).WithFields(logrus.Fields{"name": this.getName(), "panic": panic, "stack": stack}).Error("单协程池，退出")
			}

			go func() {
				this.lock.Lock()
				defer this.lock.Unlock()

				if this.taskName == name {
					this.taskName = ""
				}
			}()
		})

		task(ctx, this)
	}

	if name == "" {
		name = GenStrId()
	}
	if this.taskName == name {
		logrus.WithContext(ctx).WithFields(logrus.Fields{"name": name}).Warn("单协程池，单次任务已添加")
		return nil
	}
	this.cancel(ctx)
	ctx, this.ctxCancel = context.WithCancel(ctx)

	if CtxDone(ctx) {
		this.cancel(ctx)
		logrus.WithContext(ctx).WithFields(logrus.Fields{"name": this.getName()}).Info("单协程池，已取消")
		return nil
	}
	if this.isClose(ctx) {
		this.cancel(ctx)
		logrus.WithContext(ctx).WithFields(logrus.Fields{"name": this.getName()}).Info("单协程池，已关闭")
		return nil
	}
	err := this.pool.Submit(submit)
	if err != nil {
		this.cancel(ctx)
		logrus.WithContext(ctx).WithFields(logrus.Fields{"name": this.getName(), "err": err}).Error("单协程池，添加单次任务异常")
		return errors.Errorf("单协程池，添加单次任务异常: %+v", err)
	}
	this.taskName = name

	return nil
}

func (this *SingleGoPool) doing(ctx context.Context) bool {
	return this.getTaskName() != ""
}
func (this *SingleGoPool) Doing(ctx context.Context) bool {
	return this.doing(ctx)
}
func (this *SingleGoPool) cancel(ctx context.Context) {
	logrus.WithContext(ctx).WithFields(logrus.Fields{"name": this.getName()}).Info("单协程池，取消")
	CancelCtx(this.ctxCancel)
	this.setTaskName("")
}
func (this *SingleGoPool) Cancel(ctx context.Context) {
	this.lock.Lock()
	defer this.lock.Unlock()

	this.cancel(ctx)
}
func (this *SingleGoPool) isClose(ctx context.Context) bool {
	isClose := true
	if this.pool != nil {
		isClose = this.pool.IsClosed()
	}
	logrus.WithContext(ctx).WithFields(logrus.Fields{"name": this.getName(), "isClose": isClose}).Info("单协程池")
	return isClose
}
func (this *SingleGoPool) IsClose(ctx context.Context) bool {
	return this.isClose(ctx)
}
func (this *SingleGoPool) close(ctx context.Context) {
	logrus.WithContext(ctx).WithFields(logrus.Fields{"name": this.getName()}).Info("单协程池，关闭")
	CancelCtx(this.ctxCancel)
	this.setTaskName("")
	if this.pool != nil {
		this.pool.Release()
	}
}
func (this *SingleGoPool) Close(ctx context.Context) {
	this.lock.Lock()
	defer this.lock.Unlock()

	this.close(ctx)
}

func (this *SingleGoPool) getName() string {
	poolName := this.poolName
	taskName := this.getTaskName()
	if poolName != "" && taskName != "" {
		return fmt.Sprintf("%s_%s", poolName, taskName)
	}
	if poolName != "" {
		return poolName
	}
	if taskName != "" {
		return taskName
	}
	return "SingleGoPool"
}
func (this *SingleGoPool) GetName() string {
	return this.getName()
}
func (this *SingleGoPool) setTaskName(name string) {
	this.taskName.Store(name)
}
func (this *SingleGoPool) getTaskName() string {
	value, _ := this.taskName.Load().(string)
	return value
}
func (this *SingleGoPool) GetTaskName() string {
	return this.getTaskName()
}
func (this *SingleGoPool) GetPoolName() string {
	return this.poolName
}
