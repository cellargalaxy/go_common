package util

import (
	"fmt"
	"strconv"
	"strings"
)

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
