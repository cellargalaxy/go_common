package tool

import (
	"context"
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"github.com/cellargalaxy/go_common/model"
	"github.com/cellargalaxy/go_common/util"
	"github.com/sirupsen/logrus"
	"net/url"
	"path"
	"sort"
	"strings"
)

func BookmarkFile2Csv(ctx context.Context, filePath string, filePaths ...string) error {
	list, err := ParseBookmarkFiles(ctx, filePaths...)
	if err != nil {
		return err
	}
	err = util.WriteCsv2FileByStruct(ctx, list, filePath)
	if err != nil {
		return err
	}
	return nil
}

func ParseBookmarkFiles(ctx context.Context, filePaths ...string) ([]model.Bookmark, error) {
	bookmarkMap := make(map[string]model.Bookmark)
	for i := range filePaths {
		object, err := ParseBookmarkFile(ctx, filePaths[i])
		if err != nil {
			return nil, err
		}
		for j := range object {
			key := object[j].Sort + object[j].Name + object[j].Url
			bookmarkMap[key] = object[j]
		}
	}

	var list []model.Bookmark
	for s := range bookmarkMap {
		list = append(list, bookmarkMap[s])
	}
	sort.Sort(model.Bookmarks(list))

	logrus.WithContext(ctx).WithFields(logrus.Fields{"len(list)": len(list)}).Info("解析书签")
	return list, nil
}

func ParseBookmarkFile(ctx context.Context, filePath string) ([]model.Bookmark, error) {
	fileInfo := util.GetFileInfo(ctx, filePath)
	if fileInfo == nil {
		logrus.WithContext(ctx).WithFields(logrus.Fields{"filePath": filePath}).Error("解析书签，文件不存在")
		return nil, fmt.Errorf("解析书签，文件不存在")
	}
	data, err := util.ReadFileWithString(ctx, filePath, "")
	if err != nil {
		return nil, err
	}
	list, err := ParseBookmark(ctx, data)
	if err != nil {
		return nil, err
	}
	return list, nil
}

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
		u, err := url.Parse(list[i].Url)
		if err != nil {
			logrus.WithContext(ctx).WithFields(logrus.Fields{"err": err}).Error("解析书签，Url非法")
			return nil, fmt.Errorf("解析书签，Url非法")
		}
		host := u.Host
		paths := strings.SplitN(list[i].Url, host, 2)
		if len(paths) != 2 {
			logrus.WithContext(ctx).WithFields(logrus.Fields{"url": list[i].Url}).Error("解析书签，Url非法")
			return nil, fmt.Errorf("解析书签，Url非法")
		}
		host = util.ReverseString(host)
		list[i].Key = host + list[i].Sort + paths[1]
	}

	sort.Sort(model.Bookmarks(list))

	logrus.WithContext(ctx).WithFields(logrus.Fields{"len(list)": len(list)}).Info("解析书签")
	return list, nil
}

func parseBookmark(ctx context.Context, doc *goquery.Document, selecter string, parentSort string) []model.Bookmark {
	var list []model.Bookmark

	aSelecter := selecter + " > a"
	doc.Find(aSelecter).Each(func(i int, a *goquery.Selection) {
		var object model.Bookmark
		object.Sort = parentSort
		object.Name = a.Text()
		object.Url, _ = a.Attr("href")
		if object.Url == "" {
			return
		}
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
