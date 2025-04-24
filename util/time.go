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
var E8Loc = time.FixedZone("GMT", 8*3600)
var UTCLoc = time.UTC

func ParseStr2Time(ctx context.Context, layout, value string, loc *time.Location) (time.Time, error) {
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
	return date.UnixNano() / 1e6
}

func MsTs2Time(ts int64) time.Time {
	return time.Unix(0, ts*1e6)
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
