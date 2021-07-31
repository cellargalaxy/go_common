package util

import (
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

//斜率,截距
func LeastSquares(points [][]float64) (float64, float64) {
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
	k = ResetNanInf(k)
	a = ResetNanInf(a)
	return k, a
}

//max,min
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

//max,min
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

func Abs(value float64) float64 {
	if value >= 0 {
		return value
	}
	return -value
}
