package main

import (
	"github.com/cellargalaxy/go_common/consd"
	"github.com/cellargalaxy/go_common/model"
	"github.com/cellargalaxy/go_common/util"
	"time"
)

func init() {
	consd.Init()
	model.Init()
	util.Init()
}

func main() {
	time.Sleep(time.Hour)
}
