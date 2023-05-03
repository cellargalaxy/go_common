package util

import (
	"testing"
)

func TestStruct(t *testing.T) {
	ctx := GenCtx()

	var list []int

	list = Intersection(ctx, []int{1, 2, 3}, []int{3, 4, 5})
	if len(list) != 1 || list[0] != 3 {
		t.Errorf(`if len(list) != 1 || list[0] != 3 {`)
		return
	}

	list = DifferenceSet(ctx, []int{1, 2, 3}, []int{3, 4, 5})
	if len(list) != 2 || Sum(list...) != 3 {
		t.Errorf(`if len(list) != 5 || Sum(list...) != 15 {`)
		return
	}

	list = Distinct(ctx, 1, 2, 2, 3)
	if len(list) != 3 || Sum(list...) != 6 {
		t.Errorf(`if len(list) != 3 || Sum(list...) != 6 {`)
		return
	}
}
