package util

import (
	"math"
	"testing"
)

func TestMath(t *testing.T) {
	if ResetNanInf(math.NaN()) != 0 {
		t.Errorf(`if ResetNanInf(math.NaN()) != 0 {`)
		return
	}

	max, min := MaxAndMin(4, 2, 1, 4, 9, 7, 5, 3, 1, 1, 2, 4)
	if max != 9 {
		t.Errorf(`if max != 9 {`)
		return
	}
	if min != 1 {
		t.Errorf(`if min != 1 {`)
		return
	}

	max = Max(1, 4, 2, 7, 5, 2, 4)
	if max != 7 {
		t.Errorf(`if max != 7 {`)
		return
	}

	min = Min(4, 2, 7, 5, 2, 4)
	if min != 2 {
		t.Errorf(`if min != 2 {`)
		return
	}

	if Abs(-4514) != 4514 {
		t.Errorf(`if Abs(-4514) != 4514 {`)
		return
	}

	if Sum(1, 2, 3, 4, 5, 6, 7, 8, 9) != 45 {
		t.Errorf(`if Sum(1, 2, 3, 4, 5, 6, 7, 8, 9) != 25 {`)
		return
	}

	if Avg(1, 2, 3, 4, 5, 6, 7, 8, 9) != 5 {
		t.Errorf(`if Avg(1, 2, 3, 4, 5, 6, 7, 8, 9) != 5 {`)
		return
	}

	k, c := LeastSquare([2]int{1, 2}, [2]int{2, 3})
	if k != 1 {
		t.Errorf(`if k != 1 {`)
		return
	}
	if c != 1 {
		t.Errorf(`if c != 1 {`)
		return
	}

	avg, svar := AvgAndSVar(1, 2, 3, 4, 5, 6, 7, 8, 9)
	if avg != 5 {
		t.Errorf(`if avg != 5 {`)
		return
	}
	if svar < 2.58198889747161 || 2.58198889747162 < svar {
		t.Errorf(`if svar < 2.58198889747161 || 2.58198889747162 < svar {`)
		return
	}

	if FloatRoundInt(123.456, 2) != 12346 {
		t.Errorf(`if FloatRoundInt(123.456, 2) != 12346 {`)
		return
	}

	if IntDivFloat(12346, 100.0) != 123.46 {
		t.Errorf(`if IntDivFloat(12346, 100) != 123.46 {`)
		return
	}
}
