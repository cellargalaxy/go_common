package tool

import (
	"context"
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"github.com/cellargalaxy/go_common/model"
	"github.com/cellargalaxy/go_common/util"
	"github.com/sirupsen/logrus"
	"path"
	"strings"
)

func ParseBookmark(ctx context.Context, data string) ([]model.Bookmark, error) {
	data = strings.ReplaceAll(data, "<p>", "")

	var err error

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(data))
	if err != nil {
		logrus.WithContext(ctx).WithFields(logrus.Fields{"err": err}).Error("解析书签，解析异常")
		return nil, err
	}

	list := parseBookmark(ctx, doc, "body > dl > dt", "")
	for i := range list {
		fmt.Println(util.ToJsonString(list[i]))
	}

	logrus.WithContext(ctx).WithFields(logrus.Fields{"list": nil}).Info("天天基金基金概况")
	return nil, err
}

func parseBookmark(ctx context.Context, doc *goquery.Document, selecter string, parentSort string) []model.Bookmark {
	var list []model.Bookmark

	aSelecter := selecter + " > a"
	doc.Find(aSelecter).Each(func(i int, a *goquery.Selection) {
		var object model.Bookmark
		object.Sort = parentSort
		object.Name = a.Text()
		object.Url, _ = a.Attr("href")
		object.Icon, _ = a.Attr("icon")
		list = append(list, object)
	})
	if len(list) > 0 {
		return list
	}

	h3Selecter := selecter + " > h3"
	var sort string
	doc.Find(h3Selecter).Each(func(i int, h3 *goquery.Selection) {
		sort = h3.Text()
	})
	if sort == "" {
		return list
	}
	sort = path.Join(parentSort, sort)

	dtSelecter := selecter + " > dl > dt"
	doc.Find(dtSelecter).Each(func(i int, a *goquery.Selection) {
		dtSelecterI := dtSelecter + fmt.Sprintf(":nth-child(%+v)", i+1)
		object := parseBookmark(ctx, doc, dtSelecterI, sort)
		list = append(list, object...)
	})

	return list
}
