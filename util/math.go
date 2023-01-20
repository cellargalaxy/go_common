package util

import (
	"context"
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
func MaxAndMins(data []float64) (float64, float64) {
	if len(data) == 0 {
		return 0, 0
	}
	max := data[0]
	min := data[0]
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

// max,min
func MaxAndMin(data ...float64) (float64, float64) {
	return MaxAndMins(data)
}

func Maxs(data []float64) float64 {
	if len(data) == 0 {
		return 0
	}
	max := data[0]
	for i := range data {
		if max < data[i] {
			max = data[i]
		}
	}
	return max
}
func Max(data ...float64) float64 {
	return Maxs(data)
}

func Mins(data []float64) float64 {
	if len(data) == 0 {
		return 0
	}
	min := data[0]
	for i := range data {
		if data[i] < min {
			min = data[i]
		}
	}
	return min
}

func Min(data ...float64) float64 {
	return Mins(data)
}

func AbsFloat64(value float64) float64 {
	return math.Abs(value)
}

func AbsInt64(value int64) int64 {
	if value >= 0 {
		return value
	}
	return -value
}

func AbsInt32(value int32) int32 {
	if value >= 0 {
		return value
	}
	return -value
}

func AbsInt(value int) int {
	if value >= 0 {
		return value
	}
	return -value
}

func Avg(data ...float64) float64 {
	return Avgs(data)
}

func Avgs(data []float64) float64 {
	if len(data) <= 0 {
		return 0
	}
	var avg float64
	for i := range data {
		avg += data[i]
	}
	return avg / float64(len(data))
}

func AvgAndSVar(data []float64) (float64, float64) {
	avg, variance := AvgAndVar(data)
	return avg, math.Pow(variance, 0.5)
}

func AvgAndVar(data []float64) (float64, float64) {
	if len(data) <= 0 {
		return 0, 0
	}
	avg := Avgs(data)
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

// 四舍五入
func Float64RoundInt64(value float64) int64 {
	return int64(math.Floor(value + 0.5))
}
