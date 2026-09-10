package main

import (
	"time"

	"github.com/cellargalaxy/go_common/util"
	"github.com/sirupsen/logrus"
)

func init() {
	util.Init("go_common")
}

func main() {
	ctx := util.GenCtx()
	for i := 0; i < 100; i++ {
		logrus.WithContext(ctx).WithFields(logrus.Fields{"id": util.GenStrId()}).Info("打印日志")
		util.Sleep(ctx, time.Second)
	}
}
