package test

import (
	"github.com/cellargalaxy/go_common/tool"
	"github.com/cellargalaxy/go_common/util"
	"testing"
)

/*
1. 下载浏览器书签 -> bookmark.html
2. bookmark.html -> bookmark_new.csv

3. 拉取github的bookmark_en.txt
4. bookmark_en.txt -> bookmark_back.csv

5. 比较bookmark_new.csv和bookmark_back.csv

6. bookmark_back.csv -> bookmark.html
7. 上传到浏览器里

8. bookmark_back.csv -> bookmark_en.txt
9. bookmark_en.txt推到github
*/

// 2. bookmark.html -> bookmark_new.csv
func TestBookmark(t *testing.T) {
	ctx := util.GenCtx()
	err := tool.BookmarkFile2Csv(ctx, "bookmark_new.csv",
		"bookmark.html",
	)
	if err != nil {
		panic(err)
	}
}

// 4. bookmark_en.txt -> bookmark_back.csv
func TestDeAesCbcBookmark(t *testing.T) {
	ctx := util.GenCtx()
	secret, err := util.ReadFile2String(ctx, "bookmark_secret.txt", "")
	if err != nil {
		panic(err)
	}
	if secret == "" {
		panic("secret为空")
	}
	en, err := util.ReadFile2String(ctx, "bookmark_en.txt", "")
	if err != nil {
		panic(err)
	}
	text, err := util.DeAesCbcString(ctx, en, secret)
	if err != nil {
		panic(err)
	}
	err = util.WriteString2File(ctx, text, "bookmark_back.csv")
	if err != nil {
		panic(err)
	}
}

// 6. bookmark_back.csv -> bookmark.html
func TestBookmarkCsv2Xml(t *testing.T) {
	ctx := util.GenCtx()
	tool.BookmarkCsv2Xml(ctx, "bookmark_back.csv", "bookmark.html")
}

// 8. bookmark_back.csv -> bookmark_en.txt
func TestEnAesCbcBookmark(t *testing.T) {
	ctx := util.GenCtx()
	secret, err := util.ReadFile2String(ctx, "bookmark_secret.txt", "")
	if err != nil {
		panic(err)
	}
	if secret == "" {
		panic("secret为空")
	}
	text, err := util.ReadFile2String(ctx, "bookmark_back.csv", "")
	if err != nil {
		panic(err)
	}
	en, err := util.EnAesCbcString(ctx, text, secret)
	if err != nil {
		panic(err)
	}
	err = util.WriteString2File(ctx, en, "bookmark_en.txt")
	if err != nil {
		panic(err)
	}
}

func TestLog2Csv(t *testing.T) {
	ctx := util.GenCtx()
	tool.Log2Csv(ctx, `/home/meltsprout/code/mmm/log/mmm_job/tmp.log`, "log.csv")
}
