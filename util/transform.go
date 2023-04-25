package util

import (
	"fmt"
	"strconv"
	"strings"
)

func Interface2String(value interface{}) string {
	if value == nil {
		return ""
	}
	p, ok := value.(string)
	if ok {
		return p
	}
	i64, ok := value.(int64)
	if ok {
		return Int642String(i64)
	}
	i32, ok := value.(int32)
	if ok {
		return Int642String(int64(i32))
	}
	i, ok := value.(int)
	if ok {
		return Int2String(i)
	}
	i8, ok := value.(int8)
	if ok {
		return Int2String(int(i8))
	}
	f64, ok := value.(float64)
	if ok {
		return Float642String(f64)
	}
	f32, ok := value.(float32)
	if ok {
		return Float642String(float64(f32))
	}
	return fmt.Sprint(value)
}
func Interface2Strings(value ...interface{}) []string {
	list := make([]string, 0, len(value))
	for i := range value {
		list = append(list, Interface2String(value[i]))
	}
	return list
}

func Interface2Float64(value interface{}) float64 {
	if value == nil {
		return 0
	}
	f64, ok := value.(float64)
	if ok {
		return f64
	}
	f32, ok := value.(float32)
	if ok {
		return float64(f32)
	}
	i64, ok := value.(int64)
	if ok {
		return float64(i64)
	}
	i32, ok := value.(int32)
	if ok {
		return float64(i32)
	}
	i, ok := value.(int)
	if ok {
		return float64(i)
	}
	i8, ok := value.(int8)
	if ok {
		return float64(i8)
	}
	return String2Float64(Interface2String(value))
}
func Interface2Float64s(value ...interface{}) []float64 {
	list := make([]float64, 0, len(value))
	for i := range value {
		list = append(list, Interface2Float64(value[i]))
	}
	return list
}

func Interface2Int(value interface{}) int {
	if value == nil {
		return 0
	}
	i, ok := value.(int)
	if ok {
		return i
	}
	i64, ok := value.(int64)
	if ok {
		return int(i64)
	}
	i32, ok := value.(int32)
	if ok {
		return int(i32)
	}
	i8, ok := value.(int8)
	if ok {
		return int(i8)
	}
	f64, ok := value.(float64)
	if ok {
		return int(f64)
	}
	f32, ok := value.(float32)
	if ok {
		return int(f32)
	}
	return String2Int(Interface2String(value))
}
func Interface2Ints(value ...interface{}) []int {
	list := make([]int, 0, len(value))
	for i := range value {
		list = append(list, Interface2Int(value[i]))
	}
	return list
}

func Interface2Int64(value interface{}) int64 {
	if value == nil {
		return 0
	}
	i64, ok := value.(int64)
	if ok {
		return i64
	}
	i, ok := value.(int)
	if ok {
		return int64(i)
	}
	i32, ok := value.(int32)
	if ok {
		return int64(i32)
	}
	i8, ok := value.(int8)
	if ok {
		return int64(i8)
	}
	f64, ok := value.(float64)
	if ok {
		return int64(f64)
	}
	f32, ok := value.(float32)
	if ok {
		return int64(f32)
	}
	return String2Int64(Interface2String(value))
}
func Interface2Int64s(value ...interface{}) []int64 {
	list := make([]int64, 0, len(value))
	for i := range value {
		list = append(list, Interface2Int64(value[i]))
	}
	return list
}

func String2Float64(value string) float64 {
	if strings.Contains(value, ".") {
		value = strings.TrimRight(value, "0")
		value = strings.TrimRight(value, ".")
	}
	data, _ := strconv.ParseFloat(value, 64)
	return data
}
func String2Float64s(value ...string) []float64 {
	list := make([]float64, 0, len(value))
	for i := range value {
		list = append(list, String2Float64(value[i]))
	}
	return list
}

func String2Int64(value string) int64 {
	data, _ := strconv.ParseInt(value, 10, 64)
	return data
}
func String2Int64s(value ...string) []int64 {
	list := make([]int64, 0, len(value))
	for i := range value {
		list = append(list, String2Int64(value[i]))
	}
	return list
}

func String2Int(value string) int {
	data, _ := strconv.Atoi(value)
	return data
}
func String2Ints(value ...string) []int {
	list := make([]int, 0, len(value))
	for i := range value {
		list = append(list, String2Int(value[i]))
	}
	return list
}

func Float642String(value float64) string {
	var str string
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
func Float642Strings(value ...float64) []string {
	list := make([]string, 0, len(value))
	for i := range value {
		list = append(list, Float642String(value[i]))
	}
	return list
}

func Int642String(value int64) string {
	return strconv.FormatInt(value, 10)
}
func Int642Strings(value ...int64) []string {
	list := make([]string, 0, len(value))
	for i := range value {
		list = append(list, Int642String(value[i]))
	}
	return list
}

func Int2String(value int) string {
	return strconv.Itoa(value)
}
func Int2Strings(value ...int) []string {
	list := make([]string, 0, len(value))
	for i := range value {
		list = append(list, Int2String(value[i]))
	}
	return list
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
