package main

import (
	"github.com/cellargalaxy/go_common/model"
	"github.com/cellargalaxy/go_common/util"
	"time"
)

func init() {
	model.Init()
	util.Init()
}

func main() {
	time.Sleep(time.Hour)
}
