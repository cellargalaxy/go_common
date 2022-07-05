package main

import (
	"fmt"
	"github.com/cellargalaxy/go_common/model"
	"github.com/cellargalaxy/go_common/util"
	"time"
)

func main() {
	c := util.GenCtx()
	var claims model.Claims
	jwtToken, err := util.ParseJWT(c, "", "", claims)
	if err != nil {
		panic(err)
	}
	fmt.Println(jwtToken)
	time.Sleep(time.Second)
}
