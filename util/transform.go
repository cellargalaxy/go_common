package util

import (
	"github.com/sirupsen/logrus"
	"strconv"
	"time"
)

const DateLayout_2006年01月02日 = "2006年01月02日"
const DateLayout_2006_01_02 = "2006-01-02"
const DateLayout_060102150405_0000000 = "060102150405.000000"

var beijingLoc = time.FixedZone("GMT", 8*3600)

func Parse2BeijingTime(layout, value string) (time.Time, error) {
	date, err := time.ParseInLocation(layout, value, beijingLoc)
	if err != nil {
		logrus.WithFields(logrus.Fields{"err": err}).Error("解析北京时间字符串异常")
	}
	return date, err
}

func Parse2BeijingTimestamp(layout, value string) (int64, error) {
	date, err := Parse2BeijingTime(layout, value)
	if err != nil {
		return 0, err
	}
	return date.Unix(), err
}

func String2Int64(text string) int64 {
	data, _ := strconv.ParseInt(text, 10, 64)
	return data
}

func String2Int(text string) int {
	data, _ := strconv.Atoi(text)
	return data
}
