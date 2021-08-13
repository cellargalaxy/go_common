package util

import (
	"github.com/sirupsen/logrus"
	"strconv"
	"time"
)

const DateLayout_2006年01月02日 = "2006年01月02日"
const DateLayout_2006年01月02日15点04分05秒 = "2006年01月02日 15点04分05秒"
const DateLayout_2006_01_02 = "2006-01-02"
const DateLayout_2006_01_02_15_04_05 = "2006-01-02 15:04:05"
const DateLayout_060102150405_0000000 = "060102150405.000000"

var beijingLoc = time.FixedZone("GMT", 8*3600)

func Parse2BeijingTime(layout, value string) (time.Time, error) {
	date, err := time.ParseInLocation(layout, value, beijingLoc)
	if err != nil {
		logrus.WithFields(logrus.Fields{"err": err}).Error("解析北京时间字符串异常")
	}
	return date, err
}

func Parse2BeijingTs(layout, value string) (int64, error) {
	date, err := Parse2BeijingTime(layout, value)
	if err != nil {
		return 0, err
	}
	return date.Unix(), err
}

func Time2MsTs(date time.Time) int64 {
	return date.UnixNano() / 1e6
}

func MsTs2Time(ts int64) time.Time {
	return time.Unix(0, ts*1e6)
}

func String2Float64(text string) float64 {
	data, _ := strconv.ParseFloat(text, 64)
	return data
}

func String2Int64(text string) int64 {
	data, _ := strconv.ParseInt(text, 10, 64)
	return data
}

func String2Int(text string) int {
	data, _ := strconv.Atoi(text)
	return data
}
