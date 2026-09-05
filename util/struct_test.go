package util

import (
	"testing"
)

// Intersection 求交集 A∩B
func TestIntersection(t *testing.T) {
	ctx := GenCtx()
	got := Intersection(ctx, []int{1, 2, 3}, []int{3, 4, 5})
	if len(got) != 1 || got[0] != 3 {
		t.Errorf("Intersection = %v, 期望 [3]", got)
	}
	//无交集返回空切片而非nil，便于调用方直接range
	if got = Intersection(ctx, []int{1, 2}, []int{3, 4}); len(got) != 0 {
		t.Errorf("无交集 = %v, 期望空", got)
	}
	if got == nil {
		t.Errorf("无交集应返回空切片而非nil")
	}
	//空入参
	if got = Intersection(ctx, []int{}, []int{1}); len(got) != 0 {
		t.Errorf("空a = %v", got)
	}
	if got = Intersection(ctx, []int{1}, []int{}); len(got) != 0 {
		t.Errorf("空b = %v", got)
	}
	//完全相同
	if got = Intersection(ctx, []int{1, 2}, []int{1, 2}); len(got) != 2 {
		t.Errorf("全交集 = %v", got)
	}
	//结果顺序跟随b，且b中重复元素会重复出现（锁定现状）
	got = Intersection(ctx, []int{1, 2}, []int{2, 2, 1})
	if len(got) != 3 || got[0] != 2 || got[1] != 2 || got[2] != 1 {
		t.Errorf("Intersection(b含重复) = %v, 当前实现按b顺序保留重复", got)
	}
	//字符串
	if s := Intersection(ctx, []string{"a", "b"}, []string{"b", "c"}); len(s) != 1 || s[0] != "b" {
		t.Errorf("Intersection(字符串) = %v", s)
	}
}

// DifferenceSet 求差集 A-B
func TestDifferenceSet(t *testing.T) {
	ctx := GenCtx()
	got := DifferenceSet(ctx, []int{1, 2, 3}, []int{3, 4, 5})
	if len(got) != 2 || got[0] != 1 || got[1] != 2 {
		t.Errorf("DifferenceSet = %v, 期望 [1 2]", got)
	}
	//非对称性：A-B 与 B-A 结果不同
	rev := DifferenceSet(ctx, []int{3, 4, 5}, []int{1, 2, 3})
	if len(rev) != 2 || rev[0] != 4 || rev[1] != 5 {
		t.Errorf("DifferenceSet(反向) = %v, 期望 [4 5]", rev)
	}
	//全部被减去
	if got = DifferenceSet(ctx, []int{1, 2}, []int{1, 2}); len(got) != 0 {
		t.Errorf("全减 = %v", got)
	}
	//减空集等于原集
	if got = DifferenceSet(ctx, []int{1, 2}, []int{}); len(got) != 2 {
		t.Errorf("减空集 = %v", got)
	}
	//空集减任何都为空
	if got = DifferenceSet(ctx, []int{}, []int{1}); len(got) != 0 {
		t.Errorf("空集相减 = %v", got)
	}
}

func TestDistinct(t *testing.T) {
	ctx := GenCtx()
	got := Distinct(ctx, 1, 2, 2, 3)
	if len(got) != 3 || got[0] != 1 || got[1] != 2 || got[2] != 3 {
		t.Errorf("Distinct = %v, 期望 [1 2 3] 且保持首次出现顺序", got)
	}
	//全重复
	if got = Distinct(ctx, 5, 5, 5); len(got) != 1 || got[0] != 5 {
		t.Errorf("Distinct(全重复) = %v", got)
	}
	//无重复时原样保留
	if got = Distinct(ctx, 3, 1, 2); len(got) != 3 || got[0] != 3 || got[1] != 1 || got[2] != 2 {
		t.Errorf("Distinct(无重复) = %v, 应保持原顺序", got)
	}
	//空入参
	if got = Distinct[int](ctx); len(got) != 0 {
		t.Errorf("Distinct() = %v", got)
	}
	//字符串去重保持顺序
	if s := Distinct(ctx, "b", "a", "b"); len(s) != 2 || s[0] != "b" || s[1] != "a" {
		t.Errorf("Distinct(字符串) = %v", s)
	}
}

// Contain 的真实语义是"list与object有任一交集(Any)"，而非"list包含全部object(All)"
func TestContain(t *testing.T) {
	ctx := GenCtx()
	if !Contain(ctx, []int{1, 2, 3}, 2) {
		t.Errorf("Contain([1,2,3],2) = false")
	}
	if Contain(ctx, []int{1, 2, 3}, 9) {
		t.Errorf("Contain([1,2,3],9) = true")
	}
	//Any语义：只要命中一个即为true，即使4不在list中
	if !Contain(ctx, []int{1, 2, 3}, 3, 4) {
		t.Errorf("Contain([1,2,3],3,4) = false, 当前为Any语义应为true")
	}
	//全不命中
	if Contain(ctx, []int{1, 2, 3}, 8, 9) {
		t.Errorf("Contain([1,2,3],8,9) = true")
	}
	//空object：无可命中项，为false
	if Contain(ctx, []int{1, 2, 3}) {
		t.Errorf("Contain(空object) = true, 期望 false")
	}
	//空list
	if Contain(ctx, []int{}, 1) {
		t.Errorf("Contain(空list) = true")
	}
	if s := Contain(ctx, []string{"a", "b"}, "b"); !s {
		t.Errorf("Contain(字符串) = false")
	}
}

func TestList2Map(t *testing.T) {
	ctx := GenCtx()
	got := List2Map(ctx, 1, 2, 2, 3)
	if len(got) != 3 || !got[1] || !got[2] || !got[3] {
		t.Errorf("List2Map = %v, 期望3个键均为true", got)
	}
	//不存在的键返回零值false
	if got[99] {
		t.Errorf("不存在的键应为false")
	}
	//空入参返回空map而非nil，可直接读
	empty := List2Map[int](ctx)
	if empty == nil {
		t.Errorf("List2Map() 应返回空map而非nil")
	}
	if len(empty) != 0 || empty[1] {
		t.Errorf("List2Map() = %v", empty)
	}
}

func TestList2MapV2(t *testing.T) {
	ctx := GenCtx()
	type item struct {
		Id   int
		Name string
	}
	list := []item{{1, "a"}, {2, "b"}}
	got := List2MapV2(ctx, list, func(o item) int { return o.Id })
	if len(got) != 2 || got[1].Name != "a" || got[2].Name != "b" {
		t.Errorf("List2MapV2 = %v", got)
	}
	//键冲突时后者覆盖前者
	dup := []item{{1, "first"}, {1, "second"}}
	got = List2MapV2(ctx, dup, func(o item) int { return o.Id })
	if len(got) != 1 || got[1].Name != "second" {
		t.Errorf("键冲突 = %v, 期望后者覆盖", got)
	}
	//空列表
	if e := List2MapV2(ctx, []item{}, func(o item) int { return o.Id }); len(e) != 0 {
		t.Errorf("空列表 = %v", e)
	}
}

// V3 与 V2 的关键差异：键冲突时聚合成切片而非覆盖
func TestList2MapV3(t *testing.T) {
	ctx := GenCtx()
	type item struct {
		Group string
		Name  string
	}
	list := []item{{"a", "x"}, {"b", "y"}, {"a", "z"}}
	got := List2MapV3(ctx, list, func(o item) string { return o.Group })
	if len(got) != 2 {
		t.Fatalf("List2MapV3 分组数 = %d, 期望 2", len(got))
	}
	if len(got["a"]) != 2 || got["a"][0].Name != "x" || got["a"][1].Name != "z" {
		t.Errorf("分组a = %v, 期望聚合[x z]且保持顺序", got["a"])
	}
	if len(got["b"]) != 1 {
		t.Errorf("分组b = %v", got["b"])
	}
	if e := List2MapV3(ctx, []item{}, func(o item) string { return o.Group }); len(e) != 0 {
		t.Errorf("空列表 = %v", e)
	}
}

func TestList2List(t *testing.T) {
	ctx := GenCtx()
	type item struct {
		Id   int
		Name string
	}
	list := []item{{1, "a"}, {2, "b"}}
	//类型转换映射
	names := List2List(ctx, list, func(o item) string { return o.Name })
	if len(names) != 2 || names[0] != "a" || names[1] != "b" {
		t.Errorf("List2List = %v", names)
	}
	//顺序必须保持
	ids := List2List(ctx, list, func(o item) int { return o.Id })
	if ids[0] != 1 || ids[1] != 2 {
		t.Errorf("List2List 顺序错误 = %v", ids)
	}
	//空列表返回空切片而非nil
	empty := List2List(ctx, []item{}, func(o item) string { return o.Name })
	if empty == nil || len(empty) != 0 {
		t.Errorf("空列表 = %v", empty)
	}
	//重复元素不去重
	dup := List2List(ctx, []item{{1, "a"}, {1, "a"}}, func(o item) string { return o.Name })
	if len(dup) != 2 {
		t.Errorf("List2List 不应去重, got %v", dup)
	}
}
