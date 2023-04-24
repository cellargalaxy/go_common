package main

import (
	"fmt"
	"github.com/cellargalaxy/go_common/util"
	"github.com/sirupsen/logrus"
)

func init() {
	util.Init("go_common")
}

func main() {
	ctx := util.GenCtx()
	fmt.Println(util.GetLogIdString(ctx))
	logrus.WithContext(ctx).WithFields(logrus.Fields{}).Info("打印日志")
}
