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

var randRunes = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

func GenRandStr(n int) string {
	if n <= 0 {
		return ""
	}
	b := make([]rune, n)
	for i := range b {
		b[i] = randRunes[rand.Intn(len(randRunes))]
	}
	return string(b)
}

func GenIdByTime(time time.Time) int64 {
	str := time.In(E8Loc).Format(DateLayout_060102150405_0000)
	str = str[:12] + str[13:]
	return Str2Int[int64](str)
}

var genIdLock struct {
	sync.Mutex
	tick int64
}

func GenId() int64 {
	genIdLock.Lock()
	defer genIdLock.Unlock()

	const IdTickMicro = 100
	tick := time.Now().UnixMicro() / IdTickMicro
	if tick <= genIdLock.tick {
		tick = genIdLock.tick + 1
	}
	genIdLock.tick = tick
	return GenIdByTime(time.UnixMicro(tick * IdTickMicro))
}
func GenStrId() string {
	return Int2Str(GenId())
}
func ParseId(ctx context.Context, id int64) (time.Time, error) {
	return ParseStrId(ctx, Int2Str(id))
}
func ParseStrId(ctx context.Context, id string) (time.Time, error) {
	const IdLen = 16
	if 0 < len(id) && len(id) < IdLen {
		id = strings.Repeat("0", IdLen-len(id)) + id
	}
	if len(id) != IdLen {
		logrus.WithContext(ctx).WithFields(logrus.Fields{}).Warn("解析ID，非法长度ID")
		return time.Time{}, errors.Errorf("解析ID，非法长度ID")
	}
	id = id[:12] + "." + id[12:]
	return ParseStr2Time(ctx, DateLayout_060102150405_0000, id, E8Loc)
}
