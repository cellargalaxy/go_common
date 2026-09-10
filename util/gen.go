package util

import (
	"context"
	"math/rand"
	"strings"
	"sync"
	"time"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

func GenRandStr(n int) string {
	if n <= 0 {
		return ""
	}
	var runes = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")
	b := make([]rune, n)
	for i := range b {
		b[i] = runes[rand.Intn(len(runes))]
	}
	return string(b)
}

func GenIdByTime(time time.Time) int64 {
	str := time.In(E8Loc).Format(DateLayout_060102150405_0000000)
	str = str[:12] + str[13:]
	return Str2Int[int64](str)
}

var genIdLock struct {
	sync.Mutex //这里不使用指针会有问题吗
	micro      int64
}

func GenId() int64 {
	genIdLock.Lock()
	defer genIdLock.Unlock()
	now := time.Now()
	micro := now.UnixMicro()
	if micro <= genIdLock.micro {
		micro = genIdLock.micro + 1
	}
	return GenIdByTime(time.UnixMicro(micro))
}
func GenStrId() string {
	return Int2Str(GenId())
}
func ParseId(ctx context.Context, id int64) (time.Time, error) {
	return ParseStrId(ctx, Int2Str(id))
}
func ParseStrId(ctx context.Context, id string) (time.Time, error) {
	if 0 < len(id) && len(id) < 18 {
		id = strings.Repeat("0", 18-len(id)) + id
	}
	if len(id) != 18 {
		logrus.WithContext(ctx).WithFields(logrus.Fields{}).Warn("解析ID，非法长度ID")
		return time.Time{}, errors.Errorf("解析ID，非法长度ID")
	}
	id = id[:12] + "." + id[12:]
	return ParseStr2Time(ctx, DateLayout_060102150405_0000000, id, E8Loc)
}
