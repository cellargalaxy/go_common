package util

import (
	"strconv"
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
