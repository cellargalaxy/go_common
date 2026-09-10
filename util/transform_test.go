package util

import (
	"math"
	"testing"
)

// Int2Str：必须覆盖无符号大值，此前实现用 strconv.Itoa(int(v)) 会把 uint64 最大值回绕成 -1
func TestInt2Str(t *testing.T) {
	if got := Int2Str(0); got != "0" {
		t.Errorf(`Int2Str(0) = %q, 期望 "0"`, got)
	}
	if got := Int2Str(-42); got != "-42" {
		t.Errorf(`Int2Str(-42) = %q, 期望 "-42"`, got)
	}
	if got := Int2Str(int64(math.MaxInt64)); got != "9223372036854775807" {
		t.Errorf(`Int2Str(MaxInt64) = %q`, got)
	}
	if got := Int2Str(int64(math.MinInt64)); got != "-9223372036854775808" {
		t.Errorf(`Int2Str(MinInt64) = %q`, got)
	}
	//无符号最大值：回绕缺陷的回归哨兵
	if got := Int2Str(uint64(math.MaxUint64)); got != "18446744073709551615" {
		t.Errorf(`Int2Str(MaxUint64) = %q, 期望 "18446744073709551615"（回绕缺陷复现）`, got)
	}
	if got := Int2Str(uint32(math.MaxUint32)); got != "4294967295" {
		t.Errorf(`Int2Str(MaxUint32) = %q`, got)
	}
	//本库18位ID必须精确
	if got := Int2Str(int64(260904172648391503)); got != "260904172648391503" {
		t.Errorf(`Int2Str(18位ID) = %q`, got)
	}
	if got := Int2Strs(1, -2, 3); len(got) != 3 || got[0] != "1" || got[1] != "-2" || got[2] != "3" {
		t.Errorf("Int2Strs = %v", got)
	}
	if got := Int2Strs[int](); len(got) != 0 {
		t.Errorf("Int2Strs() 空入参应返回空切片, got %v", got)
	}
}

// Str2Int：须容忍空格，且不得在解析阶段截断64位精度
func TestStr2Int(t *testing.T) {
	cases := map[string]int64{
		"0": 0, "42": 42, "-42": -42, " 42 ": 42, "\t7\n": 7,
		"9223372036854775807": math.MaxInt64,
		"260904172648391503":  260904172648391503,
		//非法输入约定返回0而非panic
		"": 0, "abc": 0, "1.5": 0, "1,000": 0, "0x10": 0,
	}
	for in, want := range cases {
		if got := Str2Int[int64](in); got != want {
			t.Errorf("Str2Int(%q) = %d, 期望 %d", in, got, want)
		}
	}
	//前导零与正号
	if got := Str2Int[int]("007"); got != 7 {
		t.Errorf(`Str2Int("007") = %d`, got)
	}
	if got := Str2Int[int]("+7"); got != 7 {
		t.Errorf(`Str2Int("+7") = %d`, got)
	}
	if got := Str2Ints[int]("1", "x", "3"); len(got) != 3 || got[0] != 1 || got[1] != 0 || got[2] != 3 {
		t.Errorf("Str2Ints = %v", got)
	}
}

// Str2Int 的溢出语义：ParseInt 溢出时返回钳制值而非 0，且实现丢弃了 error，
// 所以超出 int64 范围的输入不会得到 0。这与"非法输入返回0"的直觉相反，
// 属于调用方必须知道的边界，这里锁定现状防止无意改变。
func TestStr2IntOverflow(t *testing.T) {
	if got := Str2Int[int64]("99999999999999999999"); got != math.MaxInt64 {
		t.Errorf(`Str2Int("99999999999999999999") = %d, 期望钳制为 MaxInt64(%d)`, got, int64(math.MaxInt64))
	}
	if got := Str2Int[int64]("-99999999999999999999"); got != math.MinInt64 {
		t.Errorf(`Str2Int("-99999999999999999999") = %d, 期望钳制为 MinInt64`, got)
	}
	//窄类型按Go转换规则回绕：MaxInt64 -> int8 为 -1
	if got := Str2Int[int8]("99999999999999999999"); got != -1 {
		t.Errorf(`Str2Int[int8]("99999999999999999999") = %d, 期望 -1（MaxInt64截断至int8）`, got)
	}
	//恰好边界值不受影响
	if got := Str2Int[int64]("9223372036854775807"); got != math.MaxInt64 {
		t.Errorf("MaxInt64 边界解析错误: %d", got)
	}
}

// Str2Float：科学计数法是此前"TrimRight('0')"缺陷的重灾区，1.0e20 曾被算成 100
func TestStr2Float(t *testing.T) {
	cases := map[string]float64{
		"0": 0, "100": 100, "1.5": 1.5, "-0.5": -0.5,
		"1.500": 1.5, "0.000": 0, "12.": 12, ".5": 0.5,
		" 1.25 ": 1.25,
		//科学计数法：量级绝不能被破坏
		"1e3": 1000, "1.0e20": 1e20, "1.5e10": 1.5e10, "1.0E20": 1e20, "2.30e5": 230000,
		"-1.0e20": -1e20, "1e-3": 0.001,
		//非法输入约定返回0
		"": 0, "abc": 0, "1.2.3": 0,
	}
	for in, want := range cases {
		if got := Str2Float[float64](in); got != want {
			t.Errorf("Str2Float(%q) = %v, 期望 %v", in, got, want)
		}
	}
	if got := Str2Floats[float64]("1.5", "bad", "2.5"); len(got) != 3 || got[0] != 1.5 || got[1] != 0 || got[2] != 2.5 {
		t.Errorf("Str2Floats = %v", got)
	}
}

// Float2Str：验证其自定义的尾数收敛逻辑，并锁定已知精度边界
func TestFloat2Str(t *testing.T) {
	cases := map[float64]string{
		0: "0", 1: "1", -1: "-1", 0.1: "0.1", 0.5: "0.5", 1.5: "1.5",
		123.456: "123.456", 100: "100", -0.5: "-0.5",
		3.14159265358979: "3.14159265358979",
		//不输出科学计数法，展开为完整数字
		1e20: "100000000000000000000",
		//浮点毛刺被收敛掉，这是本函数存在的意义
		0.30000000000000004: "0.3",
	}
	for in, want := range cases {
		if got := Float2Str(in); got != want {
			t.Errorf("Float2Str(%v) = %q, 期望 %q", in, got, want)
		}
	}
	//已知边界：截断到16位小数，1e-20 会被压成 "0"（记录现状，防止无意改变）
	if got := Float2Str(1e-20); got != "0" {
		t.Errorf(`Float2Str(1e-20) = %q, 当前实现约定为 "0"`, got)
	}
	//无穷与NaN不应产生带小数点的怪异输出
	for _, v := range []float64{math.Inf(1), math.Inf(-1), math.NaN()} {
		if got := Float2Str(v); got == "" {
			t.Errorf("Float2Str(%v) 返回空串", v)
		}
	}
	//绝大多数常规值应能往返
	for _, v := range []float64{0, 1, -1, 0.1, 0.5, 1.5, 123.456, 100, -0.5, 0.7, 1.1} {
		if back := Str2Float[float64](Float2Str(v)); back != v {
			t.Errorf("往返不一致: %v -> %q -> %v", v, Float2Str(v), back)
		}
	}
	if got := Float2Strs(1.5, 2.0); len(got) != 2 || got[0] != "1.5" || got[1] != "2" {
		t.Errorf("Float2Strs = %v", got)
	}
}

// Float2Str 内部"尾部连续9则进位并截断"的收敛分支：
// 该分支只在小数展开满16位且尾部出现9时才触发，此前完全未被覆盖。
// 它是有损的（会丢掉后续有效数字），这里锁定其现状，防止无意改动或误以为可精确往返。
func TestFloat2StrCarryBranch(t *testing.T) {
	cases := map[float64]string{
		1.0 / 11.0: "0.0909090909", // 0.0909090909090909 -> 尾部9进位后截断
		1.0 / 13.0: "0.076923",     // 0.0769230769230769
		1.0 / 17.0: "0.05882353",   // 0.0588235294117647
		1.0 / 19.0: "0.052631579",  // 0.0526315789473684
		1.0 / 21.0: "0.047619",     // 0.0476190476190476
		1.0 / 23.0: "0.04347826",   // 0.0434782608695652
	}
	for in, want := range cases {
		if got := Float2Str(in); got != want {
			t.Errorf("Float2Str(%.17g) = %q, 期望 %q", in, got, want)
		}
	}

	//该分支是有损的：往返后不再等于原值，这是与常规值的关键差异
	v := 1.0 / 11.0
	if back := Str2Float[float64](Float2Str(v)); back == v {
		t.Errorf("进位分支本应有损，但 %v 往返后仍精确相等", v)
	}

	//尾部为连续9但小数位不足16位时，不进入该分支，须原样保留
	if got := Float2Str(0.99); got != "0.99" {
		t.Errorf("Float2Str(0.99) = %q, 期望 0.99（不应触发进位）", got)
	}
	//全9且满16位：无"非9后接9"的位置，故不进位，仅按16位截断
	if got := Float2Str(0.9999999999999999); got != "0.999999999999999" {
		t.Errorf("Float2Str(0.9999999999999999) = %q", got)
	}
}

// Any2Str：逐类型验证，含 fmt.Sprint 兜底路径
func TestAny2Str(t *testing.T) {
	cases := []struct {
		in   any
		want string
	}{
		{nil, ""}, {"s", "s"}, {int64(1), "1"}, {int32(2), "2"}, {int(3), "3"}, {int8(4), "4"},
		{float64(1.5), "1.5"}, {float32(2.5), "2.5"},
		//未显式覆盖的类型走 fmt.Sprint 兜底
		{true, "true"}, {uint(9), "9"},
	}
	for _, c := range cases {
		if got := Any2Str(c.in); got != c.want {
			t.Errorf("Any2Str(%v %T) = %q, 期望 %q", c.in, c.in, got, c.want)
		}
	}
	//浮点走Float2Str而非fmt.Sprint，故毛刺被收敛
	if got := Any2Str(0.30000000000000004); got != "0.3" {
		t.Errorf("Any2Str(浮点毛刺) = %q, 期望 0.3", got)
	}
	if got := Any2Strs(1, "a", nil); len(got) != 3 || got[0] != "1" || got[1] != "a" || got[2] != "" {
		t.Errorf("Any2Strs = %v", got)
	}
}

func TestAny2Int(t *testing.T) {
	cases := []struct {
		in   any
		want int
	}{
		{nil, 0}, {int64(7), 7}, {int(8), 8}, {int32(9), 9}, {int8(10), 10},
		//浮点向零截断
		{float64(1.9), 1}, {float32(2.9), 2}, {float64(-1.9), -1},
		//字符串走Str2Int
		{"42", 42}, {" 42 ", 42}, {"abc", 0},
		//未覆盖类型经 fmt.Sprint 再解析
		{uint(11), 11}, {int16(12), 12},
		//bool无法解析为数字，约定为0
		{true, 0},
	}
	for _, c := range cases {
		if got := Any2Int[int](c.in); got != c.want {
			t.Errorf("Any2Int(%v %T) = %d, 期望 %d", c.in, c.in, got, c.want)
		}
	}
	if got := Any2Ints[int](1, "2", nil); len(got) != 3 || got[0] != 1 || got[1] != 2 || got[2] != 0 {
		t.Errorf("Any2Ints = %v", got)
	}
}

func TestAny2Float(t *testing.T) {
	cases := []struct {
		in   any
		want float64
	}{
		{nil, 0}, {int64(7), 7}, {int(8), 8}, {int32(9), 9}, {int8(10), 10},
		{float64(1.5), 1.5}, {float32(2.5), 2.5},
		{"1.25", 1.25}, {"1.0e20", 1e20}, {"abc", 0},
		{uint(11), 11},
	}
	for _, c := range cases {
		if got := Any2Float[float64](c.in); got != c.want {
			t.Errorf("Any2Float(%v %T) = %v, 期望 %v", c.in, c.in, got, c.want)
		}
	}
	if got := Any2Floats[float64](1.5, "2.5"); len(got) != 2 || got[0] != 1.5 || got[1] != 2.5 {
		t.Errorf("Any2Floats = %v", got)
	}
}

func TestHump2Underscore(t *testing.T) {
	cases := map[string]string{
		"":         "",
		"UserName": "user_name",
		"userName": "user_name",
		"user":     "user",
		//连续大写逐个转换，这是当前实现的既有行为
		"ID": "i_d",
		"aB": "a_b",
		//已是下划线开头时保留
		"_A": "_a",
		//非ASCII不受影响
		"已中文A": "已中文_a",
	}
	for in, want := range cases {
		if got := Hump2Underscore(in); got != want {
			t.Errorf("Hump2Underscore(%q) = %q, 期望 %q", in, got, want)
		}
	}
}

func TestReverseStr(t *testing.T) {
	cases := map[string]string{
		"": "", "a": "a", "abc": "cba",
		//按rune反转，中文与emoji不能被拆坏
		"中文测试": "试测文中",
		"a中b":  "b中a",
		"👍ab":  "ba👍",
	}
	for in, want := range cases {
		if got := ReverseStr(in); got != want {
			t.Errorf("ReverseStr(%q) = %q, 期望 %q", in, got, want)
		}
	}
	//反转两次应还原
	for _, s := range []string{"", "abc", "中文👍测试"} {
		if got := ReverseStr(ReverseStr(s)); got != s {
			t.Errorf("反转两次未还原: %q -> %q", s, got)
		}
	}
}

// 指针与切片辅助函数：重点验证是否发生别名共享
func TestPointerHelpers(t *testing.T) {
	v := 42
	p := S2P(v)
	if p == nil || *p != 42 {
		t.Fatalf("S2P = %v", p)
	}
	//S2P须复制值，修改原变量不应影响指针内容
	v = 99
	if *p != 42 {
		t.Errorf("S2P 未复制值，原变量修改穿透: *p = %d", *p)
	}

	if got := P2S(p); got != 42 {
		t.Errorf("P2S = %d", got)
	}
	//nil指针须返回零值而非panic
	var nilP *int
	if got := P2S(nilP); got != 0 {
		t.Errorf("P2S(nil) = %d, 期望 0", got)
	}

	ps := S2Ps(1, 2, 3)
	if len(ps) != 3 {
		t.Fatalf("S2Ps 长度 = %d", len(ps))
	}
	//每个元素必须是独立指针，不能都指向同一底层数组元素
	if ps[0] == ps[1] || *ps[0] != 1 || *ps[1] != 2 || *ps[2] != 3 {
		t.Errorf("S2Ps 元素共享指针或取值错误: %v %v %v", *ps[0], *ps[1], *ps[2])
	}
	*ps[0] = 100
	if *ps[1] != 2 {
		t.Errorf("S2Ps 修改一个元素影响了另一个")
	}

	ss := P2Ss(S2P(7), nil, S2P(9))
	if len(ss) != 3 || ss[0] != 7 || ss[1] != 0 || ss[2] != 9 {
		t.Errorf("P2Ss = %v（nil应转零值）", ss)
	}
}

// CopyArray 必须真复制，否则调用方修改会穿透
func TestCopyArray(t *testing.T) {
	src := []int{1, 2, 3}
	dst := CopyArray(src...)
	if len(dst) != 3 || dst[0] != 1 || dst[2] != 3 {
		t.Fatalf("CopyArray = %v", dst)
	}
	dst[0] = 100
	if src[0] != 1 {
		t.Errorf("CopyArray 未真复制，修改穿透到源切片: src = %v", src)
	}
	if got := CopyArray[int](); len(got) != 0 {
		t.Errorf("CopyArray() 空入参 = %v", got)
	}
}
