package test

import (
	"github.com/cellargalaxy/go_common/tool"
	"github.com/cellargalaxy/go_common/util"
	"testing"
)

const BookmarkSecret = ""

func TestDeAesCbcBookmark(t *testing.T) {
	ctx := util.CreateLogCtx()
	en, err := util.ReadFileWithString(ctx, "bookmark_en.txt", "")
	if err != nil {
		panic(err)
	}
	text, err := util.DeAesCbcString(ctx, en, BookmarkSecret)
	if err != nil {
		panic(err)
	}
	err = util.WriteFileWithString(ctx, "bookmark_back.csv", text)
	if err != nil {
		panic(err)
	}
}

func TestBookmark(t *testing.T) {
	ctx := util.CreateLogCtx()
	err := tool.BookmarkFile2Csv(ctx, "bookmark_new.csv",
		"/home/meltsprout/ms.html",
	)
	if err != nil {
		panic(err)
	}
}

func TestEnAesCbcBookmark(t *testing.T) {
	ctx := util.CreateLogCtx()
	text, err := util.ReadFileWithString(ctx, "bookmark_back.csv", "")
	if err != nil {
		panic(err)
	}
	en, err := util.EnAesCbcString(ctx, text, BookmarkSecret)
	if err != nil {
		panic(err)
	}
	err = util.WriteFileWithString(ctx, "bookmark_en.txt", en)
	if err != nil {
		panic(err)
	}
}
