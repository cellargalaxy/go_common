package main

import (
	"fmt"
	"github.com/cellargalaxy/go_common/util"
	"github.com/gen2brain/go-fitz"
	"github.com/sirupsen/logrus"
)

func init() {
	util.Init("go_common")
}

func main() {
	ctx := util.GenCtx()
	fmt.Println(util.GetLogIdString(ctx))
	logrus.WithContext(ctx).WithFields(logrus.Fields{}).Info("打印日志")

	doc, err := fitz.New(fmt.Sprintf("%s/招商银行交易流水.pdf", util.GetHome()))
	if err != nil {
		panic(err)
	}
	defer doc.Close()

	pages := doc.NumPage()

	for page := 0; page < pages; page++ {
		ttt, err := doc.Text(page)
		if err != nil {
			panic(err)
		}
		fmt.Println(ttt)
	}

}
