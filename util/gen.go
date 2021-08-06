package util

import (
	"strconv"
	"time"
)

func CreateId() int64 {
	now := time.Now()
	str := now.Format(DateLayout_060102150405_0000000)
	str = str[:12] + str[13:]
	logId, _ := strconv.ParseInt(str, 10, 64)
	return logId
}
