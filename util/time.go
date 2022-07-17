package util

import (
	"context"
	"github.com/sirupsen/logrus"
	"math/rand"
	"time"
)

const DateLayout_2006年01月02日 = "2006年01月02日"
const DateLayout_2006年01月02日15点04分05秒 = "2006年01月02日 15点04分05秒"
const DateLayout_2006_01_02 = "2006-01-02"
const DateLayout_2006_01_02_15_04_05 = "2006-01-02 15:04:05"
const DateLayout_060102150405_0000000 = "060102150405.000000"

var MaxTime = time.Unix(253402271999, 0)
var beijingLoc = time.FixedZone("GMT", 8*3600)

func Parse2BeijingTime(ctx context.Context, layout, value string) (time.Time, error) {
	date, err := time.ParseInLocation(layout, value, beijingLoc)
	if err != nil {
		logrus.WithContext(ctx).WithFields(logrus.Fields{"err": err}).Error("解析北京时间字符串异常")
	}
	return date, err
}

func Parse2BeijingTs(ctx context.Context, layout, value string) (int64, error) {
	date, err := Parse2BeijingTime(ctx, layout, value)
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

func WareDuration(duration time.Duration) time.Duration {
	rate := 1 - (rand.Float64() * rand.Float64())
	rate = 0.9 + (0.2 * rate)
	ns := int64(duration)
	ns = int64(float64(ns) * rate)
	return time.Duration(ns)
}

func MinDuration(data ...time.Duration) time.Duration {
	if len(data) == 0 {
		return 0
	}
	min := data[0]
	for i := range data {
		if data[i] < min {
			min = data[i]
		}
	}
	return min
}

func MaxDuration(data ...time.Duration) time.Duration {
	if len(data) == 0 {
		return 0
	}
	max := data[0]
	for i := range data {
		if max < data[i] {
			max = data[i]
		}
	}
	return max
}

func Sleep(ctx context.Context, sleep time.Duration) {
	select {
	case <-time.NewTicker(sleep).C:
	case <-ctx.Done():
	}
}

func SleepWare(ctx context.Context, sleep time.Duration) {
	Sleep(ctx, WareDuration(sleep))
}
