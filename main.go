package main

import (
	"fmt"
	"github.com/cellargalaxy/go_common/util"
)

func init() {
	util.Init("go_common")
}

func main() {
	fmt.Println(util.GenId())
}
