package util

import (
	"fmt"
	"strconv"
	"strings"
)

func String2Float64(value string) float64 {
	data, _ := strconv.ParseFloat(value, 64)
	return data
}

func String2Int64(value string) int64 {
	data, _ := strconv.ParseInt(value, 10, 64)
	return data
}

func String2Int(value string) int {
	data, _ := strconv.Atoi(value)
	return data
}

func Float642String(value float64) string {
	return strconv.FormatFloat(value, 'f', 16, 64)
}

func Int642String(value int64) string {
	return strconv.FormatInt(value, 10)
}

func Int2String(value int) string {
	return strconv.Itoa(value)
}

func Hump2Underscore(text string) string {
	for j := 'A'; j <= 'Z'; j++ {
		text = strings.ReplaceAll(text, fmt.Sprintf("%c", j), fmt.Sprintf("_%c", j+32))
	}
	if text[0] == '_' {
		text = text[1:]
	}
	return text
}

func ReverseString(s string) string {
	runes := []rune(s)
	for from, to := 0, len(runes)-1; from < to; from, to = from+1, to-1 {
		runes[from], runes[to] = runes[to], runes[from]
	}
	return string(runes)
}
