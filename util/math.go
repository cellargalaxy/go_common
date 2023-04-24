package util

import (
	"context"
	"github.com/shopspring/decimal"
	"golang.org/x/exp/constraints"
	"math"
)

func ResetNanInf(value float64) float64 {
	if IsNanInf(value) {
		return 0
	}
	return value
}
func IsNanInf(value float64) bool {
	return math.IsNaN(value) || math.IsInf(value, 0)
}

// 斜率,截距
func LeastSquare(points [][]float64) (float64, float64) {
	if len(points) <= 1 {
		return 0, 0
	}
	var xi, x2, yi, xy float64
	for i := 0; i < len(points); i++ {
		xi += points[i][0]
		x2 += points[i][0] * points[i][0]
		yi += points[i][1]
		xy += points[i][0] * points[i][1]
	}
	length := float64(len(points))
	k := (yi*xi - xy*length) / (xi*xi - x2*length) //斜率
	a := (yi*x2 - xy*xi) / (x2*length - xi*xi)     //截距
	return k, a
}

// max,min
func MaxAndMin[T constraints.Ordered](data ...T) (T, T) {
	var min, max T
	if len(data) == 0 {
		return max, min
	}
	max = data[0]
	min = data[0]
	for i := range data {
		value := data[i]
		if max < value {
			max = value
		}
		if value < min {
			min = value
		}
	}
	return max, min
}

func Max[T constraints.Ordered](data ...T) T {
	var max T
	if len(data) == 0 {
		return max
	}
	max = data[0]
	for i := range data {
		if max < data[i] {
			max = data[i]
		}
	}
	return max
}

func Min[T constraints.Ordered](data ...T) T {
	var min T
	if len(data) == 0 {
		return min
	}
	min = data[0]
	for i := range data {
		if data[i] < min {
			min = data[i]
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

func Sum[T constraints.Integer | constraints.Float](data ...T) T {
	var sum T
	for i := range data {
		sum += data[i]
	}
	return sum
}

func Avg[T constraints.Integer | constraints.Float](data ...T) T {
	avg := Sum(data...)
	return avg / T(len(data))
}

func AvgAndSVar(data []float64) (float64, float64) {
	avg, variance := AvgAndVar(data)
	return avg, math.Pow(variance, 0.5)
}
func AvgAndVar(data []float64) (float64, float64) {
	if len(data) <= 0 {
		return 0, 0
	}
	avg := Avg(data...)
	var variance float64
	for i := range data {
		variance += math.Pow(data[i]-avg, 2)
	}
	variance /= float64(len(data))
	return avg, variance
}

func SameTickFloat64(ctx context.Context, value1, value2, tick float64) bool {
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

func SameTickFloat32(ctx context.Context, value1, value2, tick float32) bool {
	return SameTickFloat64(ctx, float64(value1), float64(value2), float64(tick))
}

/*
四舍五入保留round位小数，再乘round个10倍取整

123.456 -> 123.46 -> 12346
*/
func Float64RoundInt64(value float64, round int) int64 {
	mul := int64(1)
	for i := 0; i < round; i++ {
		mul *= 10
	}
	return decimal.NewFromFloat(value).Round(int32(round)).Mul(decimal.NewFromInt(mul)).IntPart()
}

/*
除以div，再转成浮点

12346 -> 123.46
*/
func Int64DivFloat64(value int64, div float64) float64 {
	f64, _ := decimal.NewFromInt(value).Div(decimal.NewFromFloat(div)).Float64()
	return f64
}
