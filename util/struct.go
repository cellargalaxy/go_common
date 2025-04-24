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
	listMap := make(map[T]bool, len(list))
	object := make([]T, 0, len(list))
	for i := range list {
		if listMap[list[i]] {
			continue
		}
		listMap[list[i]] = true
		object = append(object, list[i])
	}
	return object
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

func List2MapV2[T constraints.Ordered, K any](ctx context.Context, list []K, getKey func(object K) T) map[T]K {
	object := make(map[T]K, len(list))
	for i := range list {
		key := getKey(list[i])
		object[key] = list[i]
	}
	return object
}

func List2MapV3[T constraints.Ordered, K any](ctx context.Context, list []K, getKey func(object K) T) map[T][]K {
	object := make(map[T][]K, len(list))
	for i := range list {
		key := getKey(list[i])
		object[key] = append(object[key], list[i])
	}
	return object
}

func List2List[T any, K any](ctx context.Context, list []T, get func(object T) K) []K {
	object := make([]K, 0, len(list))
	for i := range list {
		object = append(object, get(list[i]))
	}
	return object
}
