package util

import (
	"context"
	"github.com/shopspring/decimal"
	"golang.org/x/exp/constraints"
	"math"
	"math/rand"
)

func IsNanInf[T constraints.Float](value T) bool {
	val := float64(value)
	return math.IsNaN(val) || math.IsInf(val, 0)
}
func ResetNanInf[T constraints.Float](value T) T {
	var result T
	if IsNanInf(value) {
		return result
	}
	result = value
	return result
}

// max,min
func MaxAndMin[T constraints.Ordered](list ...T) (T, T) {
	var min, max T
	if len(list) == 0 {
		return max, min
	}
	max = list[0]
	min = list[0]
	for i := range list {
		if max < list[i] {
			max = list[i]
		}
		if list[i] < min {
			min = list[i]
		}
	}
	return max, min
}

func Max[T constraints.Ordered](list ...T) T {
	var max T
	if len(list) == 0 {
		return max
	}
	max = list[0]
	for i := range list {
		if max < list[i] {
			max = list[i]
		}
	}
	return max
}

func Min[T constraints.Ordered](list ...T) T {
	var min T
	if len(list) == 0 {
		return min
	}
	min = list[0]
	for i := range list {
		if list[i] < min {
			min = list[i]
		}
	}
	return min
}

func Abs[T constraints.Integer | constraints.Float](value T) T {
	if value < 0 {
		return -value
	}
	return value
}

func Sum[T constraints.Integer | constraints.Float](list ...T) T {
	var sum T
	for i := range list {
		sum += list[i]
	}
	return sum
}

func Avg[T constraints.Integer | constraints.Float](list ...T) T {
	avg := Sum(list...)
	return avg / T(len(list))
}

func WareNumber[T constraints.Integer | constraints.Float](value T) T {
	ns := float64(value)
	a := rand.Float64()
	b := rand.Float64()
	d := ns * 0.1 * a * b
	if a < b {
		return value + T(d)
	}
	return value - T(d)
}

// 斜率,截距
func LeastSquare[T constraints.Integer | constraints.Float](list ...[2]T) (float64, float64) {
	if len(list) <= 1 {
		return 0, 0
	}
	var xi, x2, yi, xy T
	for i := 0; i < len(list); i++ {
		xi += list[i][0]
		x2 += list[i][0] * list[i][0]
		yi += list[i][1]
		xy += list[i][0] * list[i][1]
	}
	length := T(len(list))
	k := (yi*xi - xy*length) / (xi*xi - x2*length) //斜率
	a := (yi*x2 - xy*xi) / (x2*length - xi*xi)     //截距
	return float64(k), float64(a)
}

func AvgAndSVar[T constraints.Integer | constraints.Float](data ...T) (float64, float64) {
	avg, variance := AvgAndVar(data...)
	return avg, math.Pow(variance, 0.5)
}
func AvgAndVar[T constraints.Integer | constraints.Float](list ...T) (float64, float64) {
	if len(list) <= 0 {
		return 0, 0
	}
	avg := Avg(list...)
	var variance float64
	for i := range list {
		variance += math.Pow(float64(list[i]-avg), 2)
	}
	variance /= float64(len(list))
	return float64(avg), variance
}

func SameTick[T constraints.Integer | constraints.Float](ctx context.Context, value1, value2, tick T) bool {
	if value1 < value2 {
		value1 += tick
		return value2 < value1
	}
	if value2 < value1 {
		value2 += tick
		return value1 < value2
	}
	return true
}

/*
四舍五入保留round位小数，再乘round个10倍取整

123.456 -> 123.46 -> 12346
*/
func FloatRoundInt[Integer constraints.Integer, Float constraints.Float](value Float, round Integer) Integer {
	mul := Integer(1)
	for i := Integer(0); i < round; i++ {
		mul *= 10
	}
	result := decimal.NewFromFloat(float64(value)).Round(int32(round)).Mul(decimal.NewFromInt(int64(mul))).IntPart()
	return Integer(result)
}

/*
转成浮点，除以div

12346 -> 123.46
*/
func IntDivFloat[Integer constraints.Integer, Float constraints.Float](value Integer, div Float) Float {
	result, _ := decimal.NewFromInt(int64(value)).Div(decimal.NewFromFloat(float64(div))).Float64()
	return Float(result)
}
