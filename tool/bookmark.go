package tool

import (
	"context"
	"encoding/xml"
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"github.com/cellargalaxy/go_common/model"
	"github.com/cellargalaxy/go_common/util"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"net/url"
	"path"
	"sort"
	"strings"
)

const (
	folderType = 1
	aType      = 2
)

type Dt interface {
	GetType() int
	GetKey() string
}

type Dts []Dt

func (this Dts) Len() int {
	return len(this)
}

func (this Dts) Swap(i, j int) {
	this[i], this[j] = this[j], this[i]
}

func (this Dts) Less(i, j int) bool {
	if this[i].GetType() != this[j].GetType() {
		return this[i].GetType() < this[j].GetType()
	}
	return this[i].GetKey() < this[j].GetKey()
}

type A struct {
	XMLName xml.Name `xml:"A" json:"-"`
	Href    string   `xml:"HREF,attr"`
	Text    string   `xml:",innerxml"`
}

func newA() *A {
	return new(A)
}

type ADt struct {
	XMLName xml.Name `xml:"DT" json:"-"`
	A       *A
}

func newADt() *ADt {
	aDt := new(ADt)
	aDt.A = newA()
	return aDt
}

func (this ADt) GetType() int {
	return aType
}
func (this ADt) GetKey() string {
	return this.A.Href
}

type H3 struct {
	XMLName xml.Name `xml:"H3" json:"-"`
	Text    string   `xml:",innerxml"`
}

func newH3() *H3 {
	return new(H3)
}

type Dl struct {
	XMLName xml.Name `xml:"DL" json:"-"`
	Dts     []Dt
}

func newDl() *Dl {
	return new(Dl)
}

type FolderDt struct {
	XMLName xml.Name `xml:"DT" json:"-"`
	H3      *H3
	Dl      *Dl
}

func newFolderDt() *FolderDt {
	folderDt := new(FolderDt)
	folderDt.H3 = newH3()
	folderDt.Dl = newDl()
	return folderDt
}

func (this FolderDt) GetType() int {
	return folderType
}
func (this FolderDt) GetKey() string {
	return this.H3.Text
}

func BookmarkCsv2Xml(ctx context.Context, csvPath, xmlPath string) error {
	fileInfo := util.GetFileInfo(ctx, csvPath)
	if fileInfo == nil {
		logrus.WithContext(ctx).WithFields(logrus.Fields{"csvPath": csvPath}).Error("转换书签，文件不存在")
		return errors.Errorf("转换书签，文件不存在")
	}
	var list []model.Bookmark
	err := util.ReadCsvWithFile2Struct(ctx, csvPath, &list)
	if err != nil {
		return err
	}
	root := SetBookmark(ctx, list)
	data, err := xml.MarshalIndent(root.Dl, "", "    ")
	if err != nil {
		logrus.WithContext(ctx).WithFields(logrus.Fields{"err": err}).Error("转换书签，xml序列号异常")
		return errors.Errorf("转换书签，xml序列号异常")
	}
	err = util.WriteFileWithData(ctx, xmlPath, data)
	if err != nil {
		return err
	}
	return nil
}

func SetBookmark(ctx context.Context, bookmark []model.Bookmark) *FolderDt {
	root := newFolderDt()
	for i := range bookmark {
		setBookmark(ctx, root, bookmark[i])
	}
	return root
}

func setBookmark(ctx context.Context, root *FolderDt, bookmark model.Bookmark) {
	nodes := strings.Split(bookmark.Sort, "/")

	var folder *FolderDt
	folder = root
	for i := range nodes {
		if i == 0 {
			continue
		}
		folder = getChildFolderDt(ctx, folder, nodes[i])
	}

	aDt := newADt()
	aDt.A.Text = bookmark.Name
	aDt.A.Href = bookmark.Url
	folder.Dl.Dts = append(folder.Dl.Dts, aDt)
	sort.Sort(Dts(folder.Dl.Dts))
}

func getChildFolderDt(ctx context.Context, parent *FolderDt, node string) *FolderDt {
	for i := range parent.Dl.Dts {
		if parent.Dl.Dts[i].GetType() == folderType {
			child := parent.Dl.Dts[i].(*FolderDt)
			if child.H3.Text == node {
				return child
			}
		}
	}
	child := newFolderDt()
	child.H3.Text = node
	parent.Dl.Dts = append(parent.Dl.Dts, child)
	sort.Sort(Dts(parent.Dl.Dts))
	return child
}

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
		return nil, errors.Errorf("解析书签，文件不存在")
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
			return nil, errors.Errorf("解析书签，Url非法")
		}
		host := u.Host
		paths := strings.SplitN(list[i].Url, host, 2)
		if len(paths) != 2 {
			logrus.WithContext(ctx).WithFields(logrus.Fields{"url": list[i].Url}).Error("解析书签，Url非法")
			return nil, errors.Errorf("解析书签，Url非法")
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
