package tool

import (
	"context"
	"github.com/cellargalaxy/go_common/util"
	"github.com/gen2brain/go-fitz"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"regexp"
	"strings"
)

func CmbBankStatementPdf2Xlsx(ctx context.Context, pdfPath, xlsxPath string) error {
	doc, err := fitz.New(pdfPath)
	if err != nil {
		logrus.WithContext(ctx).WithFields(logrus.Fields{"err": err}).Error("转换招商银行账单，异常")
		return errors.Errorf("转换招商银行账单，异常: %+v", err)
	}
	defer doc.Close()

	dateRegexp, err := regexp.Compile("^\\d\\d\\d\\d-\\d\\d-\\d\\d$")
	if err != nil {
		logrus.WithContext(ctx).WithFields(logrus.Fields{"err": err}).Error("转换招商银行账单，日期正则异常")
		return errors.Errorf("转换招商银行账单，日期正则异常: %+v", err)
	}

	var head []string
	var lines [][]string
	numPage := doc.NumPage()
	for page := 0; page < numPage; page++ {
		text, err := doc.Text(page)
		if err != nil {
			logrus.WithContext(ctx).WithFields(logrus.Fields{"err": err}).Error("转换招商银行账单，读取异常")
			return errors.Errorf("转换招商银行账单，读取异常: %+v", err)
		}

		texts := strings.Split(text, "\n")
		var line []string
		for i := range texts {
			texts[i] = strings.ReplaceAll(texts[i], " ", "")
			texts[i] = strings.TrimSpace(texts[i])
			if texts[i] == "" {
				continue
			}
			if texts[i] == "合并统计" {
				break
			}
			switch texts[i] {
			case "记账日期", "货币", "交易金额", "联机余额", "交易摘要", "对手信息", "客户摘要":
				if !util.Contain(ctx, head, texts[i]) {
					head = append(head, texts[i])
				}
			}
			if dateRegexp.MatchString(texts[i]) {
				if len(line) > 0 {
					lines = append(lines, line)
				}
				line = make([]string, 0, 7)
			}
			if line == nil {
				continue
			}
			line = append(line, texts[i])
		}
		if len(line) > 0 {
			lines = append(lines, line)
		}
	}

	for i := range lines {
		if len(lines[i]) <= len(head) {
			continue
		}
		if len(lines[i]) == 8 {
			if util.String2Int[int](lines[i][6]) > 0 {
				lines[i] = []string{
					lines[i][0],
					lines[i][1],
					lines[i][2],
					lines[i][3],
					lines[i][4],
					lines[i][5] + lines[i][6],
					lines[i][7],
				}
			} else {
				lines[i] = []string{
					lines[i][0],
					lines[i][1],
					lines[i][2],
					lines[i][3],
					lines[i][4],
					lines[i][5],
					lines[i][6] + lines[i][7],
				}
			}
			continue
		}
		if len(lines[i]) == 9 {
			lines[i] = []string{
				lines[i][0],
				lines[i][1],
				lines[i][2],
				lines[i][3],
				lines[i][4],
				lines[i][5] + lines[i][6],
				lines[i][7] + lines[i][8],
			}
			continue
		}
	}

	table := util.NewTable()
	table.AppendRow(head...)
	for i := range lines {
		table.AppendRow(lines[i]...)
	}

	err = util.XlsxStrings2File(ctx, table.ListLine(), xlsxPath)
	if err != nil {
		return err
	}

	return nil
}

func IcbcBankStatementPdf2Xlsx(ctx context.Context, pdfPath, xlsxPath string) error {
	doc, err := fitz.New(pdfPath)
	if err != nil {
		logrus.WithContext(ctx).WithFields(logrus.Fields{"err": err}).Error("转换工商银行账单，异常")
		return errors.Errorf("转换工商银行账单，异常: %+v", err)
	}
	defer doc.Close()

	dateRegexp, err := regexp.Compile("^\\d\\d\\d\\d-\\d\\d-\\d\\d$")
	if err != nil {
		logrus.WithContext(ctx).WithFields(logrus.Fields{"err": err}).Error("转换招商银行账单，日期正则异常")
		return errors.Errorf("转换招商银行账单，日期正则异常: %+v", err)
	}

	head := []string{"记账日期", "货币", "交易金额", "联机余额", "交易摘要", "对手信息"}
	var lines [][]string
	numPage := doc.NumPage()
	for page := 0; page < numPage; page++ {
		text, err := doc.Text(page)
		if err != nil {
			logrus.WithContext(ctx).WithFields(logrus.Fields{"err": err}).Error("转换工商银行账单，读取异常")
			return errors.Errorf("转换工商银行账单，读取异常: %+v", err)
		}

		ls := strings.Split(text, "\n\n")
		for _, line := range ls {
			texts := strings.Split(line, "\n")
			if len(texts) < 11 {
				continue
			}
			if !dateRegexp.MatchString(texts[0]) {
				continue
			}
			var object []string
			for i := range texts {
				if i == 0 { //记账日期
					object = append(object, texts[i])
				}
				if i == 6 { //货币
					object = append(object, texts[i])
				}
				if i == 7 { //交易金额
					object = append(object, texts[i])
				}
				if i == 8 { //联机余额
					object = append(object, texts[i])
				}
				if i == 9 { //交易摘要
					object = append(object, texts[i])
				}
				if i == 10 { //对手信息
					object = append(object, texts[i])
				}
				if i > 10 { //对手信息
					object[5] += texts[i]
				}
			}
			lines = append(lines, object)
		}
	}

	table := util.NewTable()
	table.AppendRow(head...)
	for i := range lines {
		table.AppendRow(lines[i]...)
	}

	err = util.XlsxStrings2File(ctx, table.ListLine(), xlsxPath)
	if err != nil {
		return err
	}

	return nil
}
