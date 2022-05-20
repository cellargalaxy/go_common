package util

import "context"

func DistinctI64(ctx context.Context, list []int64) []int64 {
	listMap := make(map[int64]bool, len(list))
	for i := range list {
		listMap[list[i]] = true
	}
	list = make([]int64, 0, len(listMap))
	for i := range listMap {
		list = append(list, i)
	}
	return list
}

func DistinctI32(ctx context.Context, list []int32) []int32 {
	listMap := make(map[int32]bool, len(list))
	for i := range list {
		listMap[list[i]] = true
	}
	list = make([]int32, 0, len(listMap))
	for i := range listMap {
		list = append(list, i)
	}
	return list
}

func DistinctInt(ctx context.Context, list []int) []int {
	listMap := make(map[int]bool, len(list))
	for i := range list {
		listMap[list[i]] = true
	}
	list = make([]int, 0, len(listMap))
	for i := range listMap {
		list = append(list, i)
	}
	return list
}

func DistinctString(ctx context.Context, list []string) []string {
	listMap := make(map[string]bool, len(list))
	for i := range list {
		listMap[list[i]] = true
	}
	list = make([]string, 0, len(listMap))
	for i := range listMap {
		list = append(list, i)
	}
	return list
}
