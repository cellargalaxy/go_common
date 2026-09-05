package util

import (
	"errors"
	"fmt"
	"golang.org/x/exp/constraints"
	"hash/fnv"
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
	//无符号类型必须直接转换：若漏掉而落到下面的字符串兜底，
	//大于MaxInt64的值(如MaxUint64)会在解析阶段被钳制，静默丢失数值
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
	//显式用64位解析，不依赖平台int宽度：Atoi在32位平台上会按平台int宽度解析，大数值直接溢出。
	//必须按无符号再解析一次：Int2String对无符号类型用的是FormatUint，
	//而ParseInt的上限是MaxInt64，"18446744073709551615"这类合法uint64串会被钳到MaxInt64，
	//导致 String2Int(Int2String(v)) 对 v>MaxInt64 无法往返（实测MaxUint64往返得MaxInt64）。
	//注意：溢出时ParseInt/ParseUint返回的是钳制值(MaxInt64/MinInt64/MaxUint64)而非0，
	//此处丢弃err即沿用该钳制语义。
	value = strings.TrimSpace(value)
	data, err := strconv.ParseInt(value, 10, 64)
	if err == nil {
		return T(data)
	}
	//仅在"超出int64正向范围"时回退到无符号解析，其余错误(空串、非数字)保持原有归零语义。
	//这里不能无条件回退：ParseUint对负数一律报错，回退会把"-1"的钳制值MinInt64变成0。
	if errors.Is(err, strconv.ErrRange) && !strings.HasPrefix(value, "-") {
		if udata, uerr := strconv.ParseUint(value, 10, 64); uerr == nil {
			return T(udata)
		}
	}
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
	//不能对含小数点的字符串直接TrimRight("0")，科学计数法如"1.0e20"会被截成"1.0e2"，量级出错
	data, _ := strconv.ParseFloat(strings.TrimSpace(value), 64)
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
	//不能用strconv.Itoa(int(value))，无符号大值会回绕成负数（uint64最大值曾输出"-1"）
	if value < 0 {
		return strconv.FormatInt(int64(value), 10)
	}
	return strconv.FormatUint(uint64(value), 10)
}
func Int2Strings[T constraints.Integer](value ...T) []string {
	list := make([]string, 0, len(value))
	for i := range value {
		list = append(list, Int2String(value[i]))
	}
	return list
}

func String2IntWithCarry(value string, carry int) int {
	//负carry在下面 ss[1][:carry] 处会panic(slice bounds out of range)，
	//而无小数点分支里负carry又等同于0（循环不执行）；统一夹紧为0，保证两个分支语义一致
	if carry < 0 {
		carry = 0
	}
	//必须先TrimSpace：本函数按字符串切分小数位，末尾空白会被当成有效小数位占位，
	//" 123.4 "(carry=2) 会因 "4 " 已占满2位而得到 1234 而非 12340（量级差10倍）；
	//无小数点时 "  123  "(carry=2) 拼成 "  123  00"，String2Int解析失败直接归零。
	//下游 String2Int/String2Float 本身都做了TrimSpace，此处对齐同一套入参容忍度。
	value = strings.TrimSpace(value)
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
	//整数、小数两侧可能各自残留空白（如 " 123 . 4 "），需分别清理后再按位拼接
	ss[0] = strings.TrimSpace(ss[0])
	ss[1] = strings.TrimSpace(ss[1])
	for len(ss[1]) < carry {
		ss[1] += "0"
	}
	ss[1] = ss[1][:carry]
	return String2Int[int](strings.Join(ss, ""))
}

func Hump2Underscore(text string) string {
	//空字符串直接返回，避免下标越界panic
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

func ReverseString(s string) string {
	runes := []rune(s)
	for from, to := 0, len(runes)-1; from < to; from, to = from+1, to-1 {
		runes[from], runes[to] = runes[to], runes[from]
	}
	return string(runes)
}

func Integer2Integer[T, K constraints.Integer](value T) K {
	return K(value)
}
func Float2Float[T, K constraints.Float](value T) K {
	return K(value)
}
func String2String[T, K ~string](value T) K {
	return K(value)
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

func StringHash2Uint32(value string) uint32 {
	h := fnv.New32a()
	_, _ = h.Write([]byte(value))
	return h.Sum32()
}
