package util

import (
	"fmt"
	"strconv"
	"strings"
)

func String2Float64(value string) float64 {
	if strings.Contains(value, ".") {
		value = strings.TrimRight(value, "0")
		value = strings.TrimRight(value, ".")
	}
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
	var str string
	//str = fmt.Sprint(value)
	//if !strings.Contains(str, "e") {
	//	if strings.Contains(str, ".") {
	//		str = strings.TrimRight(str, "0")
	//		str = strings.TrimRight(str, ".")
	//	}
	//	ss := strings.Split(str, ".")
	//	if len(ss) == 2 && len(ss[1]) < 16 {
	//		return str
	//	}
	//}

	str = strconv.FormatFloat(value, 'f', 16, 64)
	str = strings.TrimRight(str, "0")
	str = strings.TrimRight(str, ".")
	ss := strings.Split(str, ".")
	if len(ss) != 2 {
		return str
	}
	list := strings.Split(ss[1], "")
	if len(list) < 16 {
		return str
	}
	list = list[:len(list)-1]
	for i := len(list) - 2; i >= 0; i-- {
		if list[i] != "9" && list[i+1] == "9" {
			list[i] = Int2String(String2Int(list[i]) + 1)
			list = list[:i+1]
			break
		}
	}
	for i := len(list) - 2; i >= 0; i-- {
		if list[i] != "0" && list[i+1] == "0" {
			list = list[:i+1]
			break
		}
	}
	str = fmt.Sprintf("%s.%s", ss[0], strings.Join(list, ""))
	str = strings.TrimRight(str, "0")
	str = strings.TrimRight(str, ".")
	return str
}

func Int642String(value int64) string {
	return strconv.FormatInt(value, 10)
}

func Int2String(value int) string {
	return strconv.Itoa(value)
}

func String2IntWithCarry(value string, carry int) int {
	ss := strings.Split(value, ".")
	if len(ss) == 0 || len(ss) > 2 {
		return 0
	}
	if len(ss) == 1 {
		for i := 0; i < carry; i++ {
			ss[0] += "0"
		}
		return String2Int(ss[0])
	}
	for len(ss[1]) < carry {
		ss[1] += "0"
	}
	ss[1] = ss[1][:carry]
	return String2Int(strings.Join(ss, ""))
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
