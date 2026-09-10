package util

import (
	"fmt"
	"strconv"
	"strings"

	"golang.org/x/exp/constraints"
)

func Any2Str(value any) string {
	if value == nil {
		return ""
	}
	p, ok := value.(string)
	if ok {
		return p
	}
	i64, ok := value.(int64)
	if ok {
		return Int2Str(i64)
	}
	i32, ok := value.(int32)
	if ok {
		return Int2Str(i32)
	}
	i, ok := value.(int)
	if ok {
		return Int2Str(i)
	}
	i8, ok := value.(int8)
	if ok {
		return Int2Str(i8)
	}
	f64, ok := value.(float64)
	if ok {
		return Float2Str(f64)
	}
	f32, ok := value.(float32)
	if ok {
		return Float2Str(f32)
	}
	return fmt.Sprint(value)
}
func Any2Strs(value ...any) []string {
	list := make([]string, 0, len(value))
	for i := range value {
		list = append(list, Any2Str(value[i]))
	}
	return list
}

func Any2Float[T constraints.Float](value any) T {
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
	return Str2Float[T](Any2Str(value))
}
func Any2Floats[T constraints.Float](value ...any) []T {
	list := make([]T, 0, len(value))
	for i := range value {
		list = append(list, Any2Float[T](value[i]))
	}
	return list
}

func Any2Int[T constraints.Integer](value any) T {
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
	u64, ok := value.(uint64)
	if ok {
		return T(u64)
	}
	u, ok := value.(uint)
	if ok {
		return T(u)
	}
	f64, ok := value.(float64)
	if ok {
		return T(f64)
	}
	f32, ok := value.(float32)
	if ok {
		return T(f32)
	}
	return Str2Int[T](Any2Str(value))
}
func Any2Ints[T constraints.Integer](value ...any) []T {
	list := make([]T, 0, len(value))
	for i := range value {
		list = append(list, Any2Int[T](value[i]))
	}
	return list
}

func Str2Int[T constraints.Integer](value string) T {
	value = strings.TrimSpace(value)
	data, _ := strconv.ParseInt(value, 10, 64)
	return T(data)
}
func Str2Ints[T constraints.Integer](value ...string) []T {
	list := make([]T, 0, len(value))
	for i := range value {
		list = append(list, Str2Int[T](value[i]))
	}
	return list
}

func Str2Float[T constraints.Float](value string) T {
	value = strings.TrimSpace(value)
	data, _ := strconv.ParseFloat(value, 64)
	return T(data)
}
func Str2Floats[T constraints.Float](value ...string) []T {
	list := make([]T, 0, len(value))
	for i := range value {
		list = append(list, Str2Float[T](value[i]))
	}
	return list
}

func Float2Str[T constraints.Float](value T) string {
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
			list[i] = Int2Str(Str2Int[int](list[i]) + 1)
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
func Float2Strs[T constraints.Float](value ...T) []string {
	list := make([]string, 0, len(value))
	for i := range value {
		list = append(list, Float2Str(value[i]))
	}
	return list
}

func Int2Str[T constraints.Integer](value T) string {
	if value < 0 {
		return strconv.FormatInt(int64(value), 10)
	}
	return strconv.FormatUint(uint64(value), 10)
}
func Int2Strs[T constraints.Integer](value ...T) []string {
	list := make([]string, 0, len(value))
	for i := range value {
		list = append(list, Int2Str(value[i]))
	}
	return list
}

func Hump2Underscore(text string) string {
	if text == "" {
		return text
	}
	for j := 'A'; j <= 'Z'; j++ {
		text = strings.ReplaceAll(text, fmt.Sprintf("%c", j), fmt.Sprintf("_%c", j+32))
	}
	if text[0] == '_' {
		text = text[1:]
	}
	return text
}

func ReverseStr(s string) string {
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
	list = append(list, value...)
	return list
}
