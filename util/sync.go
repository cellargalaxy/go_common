package util

import (
	"context"
	"fmt"
	"github.com/panjf2000/ants/v2"
	"github.com/sirupsen/logrus"
	"time"
)

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
		pool.Release()
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
					pool.Release()
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
		pool.Release()
		logrus.WithContext(ctx).WithFields(logrus.Fields{"name": name, "err": err}).Error("创建常驻单协程池，添加任务异常")
		return nil, fmt.Errorf("创建常驻单协程池，添加任务异常: %+v", err)
	}
	return pool, nil
}
