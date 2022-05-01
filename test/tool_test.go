package test

import (
	"github.com/cellargalaxy/go_common/tool"
	"github.com/cellargalaxy/go_common/util"
	"testing"
)

func init() {
	util.InitDefaultLog("go_common")
}

func TestBookmark(ttt *testing.T) {
	ctx := util.CreateLogCtx()
	data, err := util.ReadFileWithString(ctx, "/home/meltsprout/ms（复件）.html", "")
	if err != nil {
		panic(err)
	}
	tool.ParseBookmark(ctx, data)
}
