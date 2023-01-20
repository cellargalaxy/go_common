package util

import (
	"context"
)

// 求交集：A∩B
func IntersectionI64(ctx context.Context, a, b []int64) []int64 {
	amap := List2MapI64(ctx, a...)
	list := make([]int64, 0, len(b))
	for i := range b {
		if !amap[b[i]] {
			continue
		}
		list = append(list, b[i])
	}
	return list
}

// 求交集：A∩B
func IntersectionI32(ctx context.Context, a, b []int32) []int32 {
	amap := List2MapI32(ctx, a...)
	list := make([]int32, 0, len(b))
	for i := range b {
		if !amap[b[i]] {
			continue
		}
		list = append(list, b[i])
	}
	return list
}

// 求交集：A∩B
func IntersectionString(ctx context.Context, a, b []string) []string {
	amap := List2MapString(ctx, a...)
	list := make([]string, 0, len(b))
	for i := range b {
		if !amap[b[i]] {
			continue
		}
		list = append(list, b[i])
	}
	return list
}

// 求差集：A-B
func DifferenceSetI64(ctx context.Context, a, b []int64) []int64 {
	bmap := List2MapI64(ctx, b...)
	list := make([]int64, 0, len(a))
	for i := range a {
		if bmap[a[i]] {
			continue
		}
		list = append(list, a[i])
	}
	return list
}

// 求差集：A-B
func DifferenceSetString(ctx context.Context, a, b []string) []string {
	bmap := List2MapString(ctx, b...)
	list := make([]string, 0, len(a))
	for i := range a {
		if bmap[a[i]] {
			continue
		}
		list = append(list, a[i])
	}
	return list
}

func DistinctI64(ctx context.Context, list []int64) []int64 {
	listMap := List2MapI64(ctx, list...)
	list = make([]int64, 0, len(listMap))
	for i := range listMap {
		list = append(list, i)
	}
	return list
}

func DistinctI32(ctx context.Context, list []int32) []int32 {
	listMap := List2MapI32(ctx, list...)
	list = make([]int32, 0, len(listMap))
	for i := range listMap {
		list = append(list, i)
	}
	return list
}

func DistinctInt(ctx context.Context, list []int) []int {
	listMap := List2MapInt(ctx, list...)
	list = make([]int, 0, len(listMap))
	for i := range listMap {
		list = append(list, i)
	}
	return list
}

func DistinctString(ctx context.Context, list []string) []string {
	listMap := List2MapString(ctx, list...)
	list = make([]string, 0, len(listMap))
	for i := range listMap {
		list = append(list, i)
	}
	return list
}

func ContainString(ctx context.Context, list []string, object string) bool {
	for i := range list {
		if list[i] == object {
			return true
		}
	}
	return false
}

func ContainInt(ctx context.Context, list []int, object int) bool {
	for i := range list {
		if list[i] == object {
			return true
		}
	}
	return false
}

func ContainInt32(ctx context.Context, list []int32, object int32) bool {
	for i := range list {
		if list[i] == object {
			return true
		}
	}
	return false
}

func ContainInt64(ctx context.Context, list []int64, object int64) bool {
	for i := range list {
		if list[i] == object {
			return true
		}
	}
	return false
}

func List2MapString(ctx context.Context, list ...string) map[string]bool {
	object := make(map[string]bool, len(list))
	for i := range list {
		object[list[i]] = true
	}
	return object
}

func List2MapI64(ctx context.Context, list ...int64) map[int64]bool {
	object := make(map[int64]bool, len(list))
	for i := range list {
		object[list[i]] = true
	}
	return object
}

func List2MapInt(ctx context.Context, list ...int) map[int]bool {
	object := make(map[int]bool, len(list))
	for i := range list {
		object[list[i]] = true
	}
	return object
}

func List2MapI32(ctx context.Context, list ...int32) map[int32]bool {
	object := make(map[int32]bool, len(list))
	for i := range list {
		object[list[i]] = true
	}
	return object
}
