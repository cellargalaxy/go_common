package util

import (
	"context"
	"github.com/sirupsen/logrus"
	"time"
)

const DateLayout_2006Y01M02D = "2006年01月02日"
const DateLayout_2006Y01M02D15H04m05S = "2006年01月02日 15点04分05秒"
const DateLayout_2006_01_02 = "2006-01-02"
const DateLayout_2006_01 = "2006-01"
const DateLayout_2006_01_02_15_04_05 = "2006-01-02 15:04:05"
const DateLayout_060102150405_0000000 = "060102150405.000000"

var TimeMax = time.Unix(253402271999, 0)
var DurationMax = 1024 * 1024 * time.Hour

// 东八区，时区名不能写GMT，否则格式化出"GMT +08:00"这种自相矛盾的结果
var E8Loc = time.FixedZone("CST", 8*3600)
var UTCLoc = time.UTC

func ParseStr2Time(ctx context.Context, layout, value string, loc *time.Location) (time.Time, error) {
	//loc为nil时 time.ParseInLocation 会panic(missing Location in call to Date)，
	//本函数是 ParseStr2Ts、ParseStr2MsTs 的公共入口，须回退到UTC而非让调用方崩溃
	if loc == nil {
		logrus.WithContext(ctx).WithFields(logrus.Fields{"layout": layout, "value": value}).Warn("解析时间字符串，时区为空，按UTC处理")
		loc = UTCLoc
	}
	date, err := time.ParseInLocation(layout, value, loc)
	if err != nil {
		logrus.WithContext(ctx).WithFields(logrus.Fields{"err": err}).Error("解析时间字符串异常")
	}
	return date, err
}

func ParseStr2Ts(ctx context.Context, layout, value string, loc *time.Location) (int64, error) {
	date, err := ParseStr2Time(ctx, layout, value, loc)
	if err != nil {
		return 0, err
	}
	return date.Unix(), err
}

func ParseStr2MsTs(ctx context.Context, layout, value string, loc *time.Location) (int64, error) {
	date, err := ParseStr2Time(ctx, layout, value, loc)
	if err != nil {
		return 0, err
	}
	return Time2MsTs(date), err
}

func Time2MsTs(date time.Time) int64 {
	//不能用UnixNano，其在1678~2262年外会int64溢出，本文件的TimeMax(9999年)即已溢出
	return date.UnixMilli()
}

func MsTs2Time(ts int64) time.Time {
	return time.UnixMilli(ts)
}

func Sleep(ctx context.Context, duration time.Duration) {
	if duration <= 0 {
		return
	}
	select {
	case <-time.After(duration):
	case <-ctx.Done():
	}
}

func SleepWare(ctx context.Context, duration time.Duration) {
	Sleep(ctx, WareNumber(duration))
}
