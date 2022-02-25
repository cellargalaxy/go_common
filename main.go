package main

import (
	"github.com/cellargalaxy/go_common/consd"
	"github.com/cellargalaxy/go_common/model"
	"github.com/cellargalaxy/go_common/test"
	"github.com/cellargalaxy/go_common/util"
)

func init() {
	consd.Init()
	model.Init()
	test.Init()
	util.Init()
}

func main() {

}
