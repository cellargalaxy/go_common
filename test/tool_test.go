package test

import (
	"github.com/cellargalaxy/go_common/tool"
	"github.com/cellargalaxy/go_common/util"
	"testing"
)

func TestDeAesCbcBookmark(t *testing.T) {
	ctx := util.CreateLogCtx()
	secret, err := util.ReadFileWithString(ctx, "bookmark_secret.txt", "")
	if err != nil {
		panic(err)
	}
	if secret == "" {
		panic("secret为空")
	}
	en, err := util.ReadFileWithString(ctx, "bookmark_en.txt", "")
	if err != nil {
		panic(err)
	}
	text, err := util.DeAesCbcString(ctx, en, secret)
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
		"/home/meltsprout/bookmarks_2022_5_2.html",
	)
	if err != nil {
		panic(err)
	}
}

func TestEnAesCbcBookmark(t *testing.T) {
	ctx := util.CreateLogCtx()
	secret, err := util.ReadFileWithString(ctx, "bookmark_secret.txt", "")
	if err != nil {
		panic(err)
	}
	if secret == "" {
		panic("secret为空")
	}
	text, err := util.ReadFileWithString(ctx, "bookmark_back.csv", "")
	if err != nil {
		panic(err)
	}
	en, err := util.EnAesCbcString(ctx, text, secret)
	if err != nil {
		panic(err)
	}
	err = util.WriteFileWithString(ctx, "bookmark_en.txt", en)
	if err != nil {
		panic(err)
	}
}

func TestBookmarkCsv2Xml(t *testing.T) {
	ctx := util.CreateLogCtx()
	tool.BookmarkCsv2Xml(ctx, "bookmark_back.csv", "bookmark.html")
}
