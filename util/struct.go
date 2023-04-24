package util

import (
	"context"
	"golang.org/x/exp/constraints"
)

// 求交集：A∩B
func Intersection[T constraints.Ordered](ctx context.Context, a, b []T) []T {
	amap := List2Map(ctx, a...)
	list := make([]T, 0, len(b))
	for i := range b {
		if !amap[b[i]] {
			continue
		}
		list = append(list, b[i])
	}
	return list
}

// 求差集：A-B
func DifferenceSet[T constraints.Ordered](ctx context.Context, a, b []T) []T {
	bmap := List2Map(ctx, b...)
	list := make([]T, 0, len(a))
	for i := range a {
		if bmap[a[i]] {
			continue
		}
		list = append(list, a[i])
	}
	return list
}

func Distinct[T constraints.Ordered](ctx context.Context, list ...T) []T {
	listMap := List2Map(ctx, list...)
	list = make([]T, 0, len(list))
	for i := range listMap {
		list = append(list, i)
	}
	return list
}

func Contain[T constraints.Ordered](ctx context.Context, list []T, object ...T) bool {
	m := List2Map(ctx, object...)
	for i := range list {
		if m[list[i]] {
			return true
		}
	}
	return false
}

func List2Map[T constraints.Ordered](ctx context.Context, list ...T) map[T]bool {
	object := make(map[T]bool, len(list))
	for i := range list {
		object[list[i]] = true
	}
	return object
}
