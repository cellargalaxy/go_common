package util

import (
	"math"
	"testing"
)

func TestIsNanInf(t *testing.T) {
	for _, v := range []float64{math.NaN(), math.Inf(1), math.Inf(-1)} {
		if !IsNanInf(v) {
			t.Errorf("IsNanInf(%v) = false, 期望 true", v)
		}
	}
	for _, v := range []float64{0, 1, -1, math.MaxFloat64, math.SmallestNonzeroFloat64} {
		if IsNanInf(v) {
			t.Errorf("IsNanInf(%v) = true, 期望 false", v)
		}
	}
}

func TestResetNanInf(t *testing.T) {
	for _, v := range []float64{math.NaN(), math.Inf(1), math.Inf(-1)} {
		if got := ResetNanInf(v); got != 0 {
			t.Errorf("ResetNanInf(%v) = %v, 期望 0", v, got)
		}
	}
	//正常值必须原样返回，不能被改动
	for _, v := range []float64{0, 1, -1, 123.456, -0.5} {
		if got := ResetNanInf(v); got != v {
			t.Errorf("ResetNanInf(%v) = %v, 期望原值", v, got)
		}
	}
	//float32 也须适用
	if got := ResetNanInf(float32(math.Inf(1))); got != 0 {
		t.Errorf("ResetNanInf(float32 Inf) = %v", got)
	}
}

func TestMaxAndMin(t *testing.T) {
	max, min := MaxAndMin(4, 2, 1, 9, 7, 5, 3)
	if max != 9 || min != 1 {
		t.Errorf("MaxAndMin = %d,%d 期望 9,1", max, min)
	}
	//单元素
	if max, min = MaxAndMin(5); max != 5 || min != 5 {
		t.Errorf("MaxAndMin(5) = %d,%d", max, min)
	}
	//空入参必须返回零值而非panic
	if max, min = MaxAndMin[int](); max != 0 || min != 0 {
		t.Errorf("MaxAndMin() = %d,%d 期望 0,0", max, min)
	}
	//全负数：min不能被零值污染
	if max, min = MaxAndMin(-5, -2, -9); max != -2 || min != -9 {
		t.Errorf("MaxAndMin(全负) = %d,%d 期望 -2,-9", max, min)
	}
	//字符串按字典序
	if smax, smin := MaxAndMin("b", "a", "c"); smax != "c" || smin != "a" {
		t.Errorf("MaxAndMin(字符串) = %q,%q", smax, smin)
	}
	//浮点
	if fmax, fmin := MaxAndMin(1.5, -2.5, 0.0); fmax != 1.5 || fmin != -2.5 {
		t.Errorf("MaxAndMin(浮点) = %v,%v", fmax, fmin)
	}
	//并列时同样保留首次出现的元素（用±0.0的符号位区分）
	if fmax, _ := MaxAndMin(0.0, math.Copysign(0, -1)); math.Signbit(fmax) {
		t.Errorf("MaxAndMin 的max在并列时未保留首次出现的值")
	}
	if _, fmin := MaxAndMin(math.Copysign(0, -1), 0.0); !math.Signbit(fmin) {
		t.Errorf("MaxAndMin 的min在并列时未保留首次出现的值")
	}
	//与独立的Max/Min必须一致
	for _, c := range [][]int{{4, 2, 9}, {-1, -5, -3}, {7}, {5, 5, 5}} {
		gotMax, gotMin := MaxAndMin(c...)
		if gotMax != Max(c...) || gotMin != Min(c...) {
			t.Errorf("MaxAndMin(%v) = %d,%d 与 Max/Min 的 %d,%d 不一致",
				c, gotMax, gotMin, Max(c...), Min(c...))
		}
	}
}

func TestMax(t *testing.T) {
	if got := Max(1, 4, 2, 7, 5); got != 7 {
		t.Errorf("Max = %d", got)
	}
	if got := Max[int](); got != 0 {
		t.Errorf("Max() 空入参 = %d, 期望 0", got)
	}
	if got := Max(5); got != 5 {
		t.Errorf("Max(5) = %d", got)
	}
	//全负数，验证不被零值初始值污染
	if got := Max(-5, -2, -9); got != -2 {
		t.Errorf("Max(全负) = %d, 期望 -2", got)
	}
	if got := Max("apple", "banana"); got != "banana" {
		t.Errorf("Max(字符串) = %q", got)
	}
	//极值边界：不能因中间变量类型不当而截断
	if got := Max(int64(math.MaxInt64), int64(0)); got != math.MaxInt64 {
		t.Errorf("Max(MaxInt64,0) = %d", got)
	}
	if got := Max(int64(math.MinInt64), int64(math.MinInt64+1)); got != math.MinInt64+1 {
		t.Errorf("Max(MinInt64近邻) = %d", got)
	}
	//最大值出现在首位/末位/中间都必须取到
	for _, c := range [][]int{{9, 1, 2}, {1, 2, 9}, {1, 9, 2}} {
		if got := Max(c...); got != 9 {
			t.Errorf("Max(%v) = %d, 期望 9", c, got)
		}
	}
	//存在并列最大值时结果仍为该值
	if got := Max(7, 3, 7); got != 7 {
		t.Errorf("Max(并列最大) = %d, 期望 7", got)
	}
	//并列时保留"首次出现"的那个：用 0.0 与 -0.0 区分（两者数值相等但符号位不同）。
	//严格 < 比较不会用后来的等值替换，故结果应保持首个 0.0 的正号
	if got := Max(0.0, math.Copysign(0, -1)); math.Signbit(got) {
		t.Errorf("Max(0.0,-0.0) 返回了后出现的等值元素，并列时应保留首次出现的值")
	}
	//浮点
	if got := Max(1.5, -2.5, 0.0); got != 1.5 {
		t.Errorf("Max(浮点) = %v", got)
	}
}

func TestMin(t *testing.T) {
	if got := Min(4, 2, 7, 5); got != 2 {
		t.Errorf("Min = %d", got)
	}
	if got := Min[int](); got != 0 {
		t.Errorf("Min() 空入参 = %d, 期望 0", got)
	}
	//全正数，验证不被零值初始值污染
	if got := Min(5, 2, 9); got != 2 {
		t.Errorf("Min(全正) = %d, 期望 2", got)
	}
	if got := Min(-5, -2, -9); got != -9 {
		t.Errorf("Min(全负) = %d", got)
	}
	if got := Min(5); got != 5 {
		t.Errorf("Min(5) = %d", got)
	}
	//极值边界
	if got := Min(int64(math.MinInt64), int64(0)); got != math.MinInt64 {
		t.Errorf("Min(MinInt64,0) = %d", got)
	}
	if got := Min(int64(math.MaxInt64), int64(math.MaxInt64-1)); got != math.MaxInt64-1 {
		t.Errorf("Min(MaxInt64近邻) = %d", got)
	}
	//最小值出现在首位/末位/中间都必须取到
	for _, c := range [][]int{{1, 5, 9}, {9, 5, 1}, {9, 1, 5}} {
		if got := Min(c...); got != 1 {
			t.Errorf("Min(%v) = %d, 期望 1", c, got)
		}
	}
	//并列最小值
	if got := Min(2, 8, 2); got != 2 {
		t.Errorf("Min(并列最小) = %d, 期望 2", got)
	}
	//并列时保留"首次出现"的那个：-0.0 与 0.0 数值相等但符号位不同，
	//严格 < 比较不会用后来的等值替换，故结果应保持首个 -0.0 的负号
	if got := Min(math.Copysign(0, -1), 0.0); !math.Signbit(got) {
		t.Errorf("Min(-0.0,0.0) 返回了后出现的等值元素，并列时应保留首次出现的值")
	}
	if got := Min("b", "a", "c"); got != "a" {
		t.Errorf("Min(字符串) = %q", got)
	}
}

func TestAbs(t *testing.T) {
	cases := map[int]int{0: 0, 5: 5, -5: 5, -4514: 4514}
	for in, want := range cases {
		if got := Abs(in); got != want {
			t.Errorf("Abs(%d) = %d, 期望 %d", in, got, want)
		}
	}
	if got := Abs(-1.5); got != 1.5 {
		t.Errorf("Abs(-1.5) = %v", got)
	}
	//已知边界：最小负整数取绝对值会溢出回自身，记录现状
	if got := Abs(int8(-128)); got != -128 {
		t.Errorf("Abs(int8 -128) = %d, 当前实现因溢出返回 -128", got)
	}
	//浮点负零：value<0 为false，故 -0.0 被原样返回，符号位仍为负。
	//数值上 -0.0 == 0 成立，此处用Signbit锁定现状，防止无意改成 <= 后行为漂移
	negZero := math.Copysign(0, -1)
	if got := Abs(negZero); got != 0 {
		t.Errorf("Abs(-0.0) = %v, 数值上应等于0", got)
	}
	if !math.Signbit(Abs(negZero)) {
		t.Errorf("Abs(-0.0) 符号位已被规范化为正，与当前实现约定不符")
	}
	//正常负浮点必须真正变正
	if got := Abs(-2.5); got != 2.5 || math.Signbit(got) {
		t.Errorf("Abs(-2.5) = %v", got)
	}
}

func TestSum(t *testing.T) {
	if got := Sum(1, 2, 3, 4, 5, 6, 7, 8, 9); got != 45 {
		t.Errorf("Sum = %d, 期望 45", got)
	}
	if got := Sum[int](); got != 0 {
		t.Errorf("Sum() = %d", got)
	}
	if got := Sum(-1, 1); got != 0 {
		t.Errorf("Sum(-1,1) = %d", got)
	}
	if got := Sum(0.5, 0.25); got != 0.75 {
		t.Errorf("Sum(浮点) = %v", got)
	}
}

// Avg：原单测用 1..9（均值恰为整数5），完全掩盖整型截断；这里用会截断的数据
func TestAvg(t *testing.T) {
	if got := Avg(1, 2, 3, 4, 5, 6, 7, 8, 9); got != 5 {
		t.Errorf("Avg(1..9) = %d, 期望 5", got)
	}
	//空入参曾导致整型除零panic
	if got := Avg[int](); got != 0 {
		t.Errorf("Avg() = %d, 期望 0（曾panic）", got)
	}
	//整型截断是该泛型签名的固有行为，明确锁定：1,2 的均值1.5被截为1
	if got := Avg(1, 2); got != 1 {
		t.Errorf("Avg(1,2) = %d, 整型入参约定截断为 1", got)
	}
	//浮点入参必须保留精度，这是与整型的关键差异
	if got := Avg(1.0, 2.0); got != 1.5 {
		t.Errorf("Avg(1.0,2.0) = %v, 期望 1.5", got)
	}
	if got := Avg(1.0, 2.0, 4.0); got < 2.333333 || got > 2.333334 {
		t.Errorf("Avg(1,2,4 浮点) = %v", got)
	}
}

// LeastSquare：原单测只用了一组恰好整数解的样本，测不出整数除法截断
func TestLeastSquare(t *testing.T) {
	//原有断言（整数精确解）必须继续成立
	k, c := LeastSquare([2]int{1, 2}, [2]int{2, 3})
	if k != 1 || c != 1 {
		t.Errorf("LeastSquare(精确解) = %v,%v 期望 1,1", k, c)
	}
	//关键：非整数解样本，真实斜率2.3、截距-0.2；整型除法缺陷会算成 2,0
	k, c = LeastSquare([2]int{0, 0}, [2]int{1, 2}, [2]int{2, 4}, [2]int{3, 7})
	if k < 2.2999 || k > 2.3001 {
		t.Errorf("LeastSquare 斜率 = %v, 期望 2.3（整数截断缺陷会得2）", k)
	}
	if c < -0.2001 || c > -0.1999 {
		t.Errorf("LeastSquare 截距 = %v, 期望 -0.2（整数截断缺陷会得0）", c)
	}
	//浮点入参应得同样结果
	kf, cf := LeastSquare([2]float64{0, 0}, [2]float64{1, 2}, [2]float64{2, 4}, [2]float64{3, 7})
	if math.Abs(kf-k) > 1e-9 || math.Abs(cf-c) > 1e-9 {
		t.Errorf("整型与浮点入参结果不一致: %v,%v vs %v,%v", k, c, kf, cf)
	}
	//样本数不足直接返回0,0
	if k, c = LeastSquare([2]int{1, 1}); k != 0 || c != 0 {
		t.Errorf("LeastSquare(单点) = %v,%v 期望 0,0", k, c)
	}
	if k, c = LeastSquare[int](); k != 0 || c != 0 {
		t.Errorf("LeastSquare() = %v,%v", k, c)
	}
}

// AvgAndVar：原单测只测了 AvgAndSVar(1..9)，均值恰为整数，掩盖了方差错一倍的缺陷
func TestAvgAndVar(t *testing.T) {
	//关键样本：整型均值1.5会被截断，导致方差从0.25错成0.5
	avg, variance := AvgAndVar(1, 2)
	if avg != 1.5 {
		t.Errorf("AvgAndVar(1,2) 均值 = %v, 期望 1.5（整型截断缺陷会得1）", avg)
	}
	if variance != 0.25 {
		t.Errorf("AvgAndVar(1,2) 方差 = %v, 期望 0.25（截断缺陷会得0.5）", variance)
	}
	//均值为整数的样本
	avg, variance = AvgAndVar(2, 4, 4, 4, 5, 5, 7, 9)
	if avg != 5 {
		t.Errorf("均值 = %v, 期望 5", avg)
	}
	if variance != 4 {
		t.Errorf("方差 = %v, 期望 4", variance)
	}
	//全相同值方差为0
	if _, variance = AvgAndVar(3, 3, 3); variance != 0 {
		t.Errorf("同值方差 = %v, 期望 0", variance)
	}
	//空入参
	if avg, variance = AvgAndVar[int](); avg != 0 || variance != 0 {
		t.Errorf("AvgAndVar() = %v,%v", avg, variance)
	}
	//单元素方差为0
	if avg, variance = AvgAndVar(7); avg != 7 || variance != 0 {
		t.Errorf("AvgAndVar(7) = %v,%v 期望 7,0", avg, variance)
	}
}

func TestAvgAndSVar(t *testing.T) {
	//原有断言必须继续成立（标准差=方差的平方根）
	avg, svar := AvgAndSVar(1, 2, 3, 4, 5, 6, 7, 8, 9)
	if avg != 5 {
		t.Errorf("AvgAndSVar 均值 = %v, 期望 5", avg)
	}
	if svar < 2.58198889747161 || svar > 2.58198889747162 {
		t.Errorf("AvgAndSVar 标准差 = %v", svar)
	}
	//标准差必须等于方差的平方根，验证两函数自洽
	_, variance := AvgAndVar(2, 4, 4, 4, 5, 5, 7, 9)
	_, svar2 := AvgAndSVar(2, 4, 4, 4, 5, 5, 7, 9)
	if math.Abs(svar2-math.Sqrt(variance)) > 1e-12 {
		t.Errorf("标准差(%v) != sqrt(方差 %v)", svar2, variance)
	}
	if _, svar = AvgAndSVar(3, 3, 3); svar != 0 {
		t.Errorf("同值标准差 = %v", svar)
	}
}

// SameTick：判断两值是否落在同一tick档位内
func TestSameTick(t *testing.T) {
	ctx := GenCtx()
	//差值小于tick即同档
	if !SameTick(ctx, 1, 2, 2) {
		t.Errorf("SameTick(1,2,tick=2) = false, 差值1<2 应为同档")
	}
	//差值等于tick不算同档
	if SameTick(ctx, 1, 3, 2) {
		t.Errorf("SameTick(1,3,tick=2) = true, 差值2不小于tick 应为不同档")
	}
	//相等值恒为同档
	if !SameTick(ctx, 5, 5, 0) {
		t.Errorf("SameTick(5,5,tick=0) = false, 相等应为同档")
	}
	//对称性：交换入参结果必须一致
	for _, c := range [][3]int{{1, 2, 2}, {1, 3, 2}, {5, 5, 1}, {10, 3, 5}} {
		a := SameTick(ctx, c[0], c[1], c[2])
		b := SameTick(ctx, c[1], c[0], c[2])
		if a != b {
			t.Errorf("SameTick 不对称: (%d,%d,tick=%d) = %v, 交换后 = %v", c[0], c[1], c[2], a, b)
		}
	}
	//浮点
	if !SameTick(ctx, 1.0, 1.5, 1.0) {
		t.Errorf("SameTick(浮点 1.0,1.5,tick=1.0) 应为同档")
	}
}

func TestFloatRoundInt(t *testing.T) {
	//注释中的示例：123.456 -> 123.46 -> 12346
	if got := FloatRoundInt(123.456, 2); got != 12346 {
		t.Errorf("FloatRoundInt(123.456,2) = %d, 期望 12346", got)
	}
	//四舍五入语义（区别于截断）
	if got := FloatRoundInt(1.005, 2); got != 101 {
		t.Errorf("FloatRoundInt(1.005,2) = %d, 期望 101（四舍五入）", got)
	}
	if got := FloatRoundInt(1.994, 2); got != 199 {
		t.Errorf("FloatRoundInt(1.994,2) = %d, 期望 199", got)
	}
	//round=0 即取整
	if got := FloatRoundInt(1.6, 0); got != 2 {
		t.Errorf("FloatRoundInt(1.6,0) = %d, 期望 2", got)
	}
	//负数
	if got := FloatRoundInt(-123.456, 2); got != -12346 {
		t.Errorf("FloatRoundInt(-123.456,2) = %d, 期望 -12346", got)
	}
	if got := FloatRoundInt(0.0, 2); got != 0 {
		t.Errorf("FloatRoundInt(0,2) = %d", got)
	}
}

func TestIntDivFloat(t *testing.T) {
	//注释中的示例：12346 -> 123.46
	if got := IntDivFloat(12346, 100.0); got != 123.46 {
		t.Errorf("IntDivFloat(12346,100) = %v, 期望 123.46", got)
	}
	if got := IntDivFloat(-12346, 100.0); got != -123.46 {
		t.Errorf("IntDivFloat(-12346,100) = %v", got)
	}
	if got := IntDivFloat(0, 100.0); got != 0 {
		t.Errorf("IntDivFloat(0,100) = %v", got)
	}
	//与 FloatRoundInt 互为逆运算
	if got := IntDivFloat(FloatRoundInt(123.46, 2), 100.0); got != 123.46 {
		t.Errorf("FloatRoundInt与IntDivFloat往返不一致: %v", got)
	}
}

// WareNumber 含随机扰动，只能验证其区间性质
func TestWareNumber(t *testing.T) {
	//扰动幅度不超过原值的10%
	for i := 0; i < 200; i++ {
		got := WareNumber(1000.0)
		if got < 900 || got > 1100 {
			t.Fatalf("WareNumber(1000) = %v, 超出±10%%区间", got)
		}
	}
	//0 扰动后仍为0
	if got := WareNumber(0.0); got != 0 {
		t.Errorf("WareNumber(0) = %v, 期望 0", got)
	}
	//应确实产生了变化（200次不可能全部相同）
	first := WareNumber(1000.0)
	allSame := true
	for i := 0; i < 200; i++ {
		if WareNumber(1000.0) != first {
			allSame = false
			break
		}
	}
	if allSame {
		t.Errorf("WareNumber 200次结果完全相同，随机扰动未生效")
	}
}
