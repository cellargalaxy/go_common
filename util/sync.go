package util

import (
	"context"
	"fmt"
	"github.com/panjf2000/ants/v2"
	"github.com/sirupsen/logrus"
)

func NewForeverSingleGoPool(ctx context.Context, task func()) (*ants.Pool, error) {
	pool, err := ants.NewPool(1, ants.WithMaxBlockingTasks(1))
	if pool == nil {
		logrus.WithContext(ctx).WithFields(logrus.Fields{}).Error("创建常驻单协程池，为空")
		return nil, fmt.Errorf("创建常驻单协程池，为空")
	}
	if err != nil {
		pool.Release()
		logrus.WithContext(ctx).WithFields(logrus.Fields{"err": err}).Error("创建常驻单协程池，异常")
		return nil, fmt.Errorf("创建常驻单协程池，异常: %+v", err)
	}

	var submit func()
	submit = func() {
		defer func() {
			go func() {
				err := pool.Submit(submit)
				if err != nil {
					logrus.WithContext(ctx).WithFields(logrus.Fields{"err": err}).Warn("创建常驻单协程池，添加任务异常")
				}
			}()
		}()

		task()
	}

	err = pool.Submit(submit)
	if err != nil {
		pool.Release()
		logrus.WithContext(ctx).WithFields(logrus.Fields{"err": err}).Error("创建常驻单协程池，添加任务异常")
		return nil, fmt.Errorf("创建常驻单协程池，添加任务异常: %+v", err)
	}
	return pool, nil
}
