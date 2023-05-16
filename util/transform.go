package util

import (
	"fmt"
	"golang.org/x/exp/constraints"
	"strconv"
	"strings"
)

func Interface2String(value any) string {
	if value == nil {
		return ""
	}
	p, ok := value.(string)
	if ok {
		return p
	}
	i64, ok := value.(int64)
	if ok {
		return Int2String(i64)
	}
	i32, ok := value.(int32)
	if ok {
		return Int2String(i32)
	}
	i, ok := value.(int)
	if ok {
		return Int2String(i)
	}
	i8, ok := value.(int8)
	if ok {
		return Int2String(i8)
	}
	f64, ok := value.(float64)
	if ok {
		return Float2String(f64)
	}
	f32, ok := value.(float32)
	if ok {
		return Float2String(f32)
	}
	return fmt.Sprint(value)
}
func Interface2Strings(value ...any) []string {
	list := make([]string, 0, len(value))
	for i := range value {
		list = append(list, Interface2String(value[i]))
	}
	return list
}

func Interface2Float[T constraints.Float](value any) T {
	if value == nil {
		return 0
	}
	i64, ok := value.(int64)
	if ok {
		return T(i64)
	}
	i, ok := value.(int)
	if ok {
		return T(i)
	}
	i32, ok := value.(int32)
	if ok {
		return T(i32)
	}
	i8, ok := value.(int8)
	if ok {
		return T(i8)
	}
	f64, ok := value.(float64)
	if ok {
		return T(f64)
	}
	f32, ok := value.(float32)
	if ok {
		return T(f32)
	}
	return String2Float[T](Interface2String(value))
}
func Interface2Floats[T constraints.Float](value ...any) []T {
	list := make([]T, 0, len(value))
	for i := range value {
		list = append(list, Interface2Float[T](value[i]))
	}
	return list
}

func Interface2Int[T constraints.Integer](value any) T {
	if value == nil {
		return 0
	}
	i64, ok := value.(int64)
	if ok {
		return T(i64)
	}
	i, ok := value.(int)
	if ok {
		return T(i)
	}
	i32, ok := value.(int32)
	if ok {
		return T(i32)
	}
	i8, ok := value.(int8)
	if ok {
		return T(i8)
	}
	f64, ok := value.(float64)
	if ok {
		return T(f64)
	}
	f32, ok := value.(float32)
	if ok {
		return T(f32)
	}
	return String2Int[T](Interface2String(value))
}
func Interface2Ints[T constraints.Integer](value ...any) []T {
	list := make([]T, 0, len(value))
	for i := range value {
		list = append(list, Interface2Int[T](value[i]))
	}
	return list
}

func String2Int[T constraints.Integer](value string) T {
	data, _ := strconv.Atoi(value)
	return T(data)
}
func String2Ints[T constraints.Integer](value ...string) []T {
	list := make([]T, 0, len(value))
	for i := range value {
		list = append(list, String2Int[T](value[i]))
	}
	return list
}

func String2Float[T constraints.Float](value string) T {
	if strings.Contains(value, ".") {
		value = strings.TrimRight(value, "0")
		value = strings.TrimRight(value, ".")
	}
	data, _ := strconv.ParseFloat(value, 64)
	return T(data)
}
func String2Floats[T constraints.Float](value ...string) []T {
	list := make([]T, 0, len(value))
	for i := range value {
		list = append(list, String2Float[T](value[i]))
	}
	return list
}

func Float2String[T constraints.Float](value T) string {
	var str string
	str = strconv.FormatFloat(float64(value), 'f', 16, 64)
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
			list[i] = Int2String(String2Int[int](list[i]) + 1)
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
func Float2Strings[T constraints.Float](value ...T) []string {
	list := make([]string, 0, len(value))
	for i := range value {
		list = append(list, Float2String(value[i]))
	}
	return list
}

func Int2String[T constraints.Integer](value T) string {
	return strconv.Itoa(int(value))
}
func Int2Strings[T constraints.Integer](value ...T) []string {
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
		return String2Int[int](ss[0])
	}
	for len(ss[1]) < carry {
		ss[1] += "0"
	}
	ss[1] = ss[1][:carry]
	return String2Int[int](strings.Join(ss, ""))
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

func S2P[T any](value T) *T {
	return &value
}
func S2Ps[T any](value ...T) []*T {
	list := make([]*T, 0, len(value))
	for i := range value {
		list = append(list, S2P(value[i]))
	}
	return list
}
func P2S[T any](value *T) T {
	var object T
	if value != nil {
		object = *value
	}
	return object
}
func P2Ss[T any](value ...*T) []T {
	list := make([]T, 0, len(value))
	for i := range value {
		list = append(list, P2S(value[i]))
	}
	return list
}
func CopyArray[T any](value ...T) []T {
	list := make([]T, 0, len(value))
	for i := range value {
		list = append(list, value[i])
	}
	return list
}
