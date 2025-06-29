package collection

import (
	"goapp/pkg/collection"
	"math/rand"
	"strings"
	"time"
)

// 找到数组中第一个满足条件的项
func First[T any](data []T, f func(*T) bool) *T {
	for _, v := range data {
		if f(&v) {
			return &v
		}
	}
	return nil
}

// 找到数组中第一个满足条件的项，如果未找到，使用默认值
func FirstOrDefault[T any](data []T, f func(*T) bool, defaultVal T) T {
	for _, v := range data {
		if f(&v) {
			return v
		}
	}
	return defaultVal
}

// 从数组中筛选出满足条件的项
func Filter[T any](data []T, filter func(*T) bool) []T {
	outArr := []T{}
	for _, item := range data {
		if filter(&item) {
			outArr = append(outArr, item)
		}
	}

	return outArr
}

// 查看数组中是否存在指定值
func Contains(data []string, target string, ignoreCase bool) bool {
	for _, v := range data {
		if ignoreCase {
			if strings.EqualFold(v, target) {
				return true
			}
		} else {
			if v == target {
				return true
			}
		}
	}
	return false
}

// 将数组中的项转换为另外一个类型的对象
func Map[TIn any, TOut any](data []TIn, f func(*TIn) (TOut, bool)) []TOut {
	outArr := []TOut{}
	for _, v := range data {
		out, ok := f(&v)
		if ok {
			outArr = append(outArr, out)
		}
	}

	return outArr
}

// 查看数组中是否存在指定条件的项
func Any[T any](data []T, condition func(*T) bool) bool {
	for _, v := range data {
		if condition(&v) {
			return true
		}
	}
	return false
}

// 检查数组是否所有的项都满足指定的条件
func All[T any](data []T, condition func(*T) bool) bool {
	for _, v := range data {
		if !condition(&v) {
			return false
		}
	}
	return true
}

// 根据指定的字段分组
func GroupBy[T any, TField comparable](data []T, fieldFilter func(*T) TField) map[TField][]T {
	out := make(map[TField][]T)
	for _, v := range data {
		f := fieldFilter(&v)
		if len(out[f]) == 0 {
			out[f] = []T{}
		}
		out[f] = append(out[f], v)
	}
	return out
}

// 根据指定的字段分组
func GroupByWithMap[S any, TOutField any, TGroupField comparable](data []S, fieldFilter func(*S) TGroupField, mapper func(*S) TOutField) map[TGroupField][]TOutField {
	out := make(map[TGroupField][]TOutField)
	for _, v := range data {
		f := fieldFilter(&v)
		if len(out[f]) == 0 {
			out[f] = []TOutField{}
		}
		out[f] = append(out[f], mapper(&v))
	}
	return out
}

// 计算满足指定条件的项的数量
func Count[T any](arr []T, condition func(*T) bool) int {
	var cnt = 0
	for _, v := range arr {
		if condition(&v) {
			cnt++
		}
	}

	return cnt
}

// 将一个长数组拆分为多个小的批次数组
func SplitIntoBatches[T any](arr []T, batchSize int) [][]T {
	batches := [][]T{}
	for i := 0; i < len(arr); i += batchSize {
		end := min(i+batchSize, len(arr))
		batches = append(batches, arr[i:end])
	}
	return batches
}

// 去除数组中重复的元素
func Deduplication[T comparable](arr []T) []T {
	set := collection.Set[T]{}
	set.Add(arr...)
	return set.ToSlice()
}

// 打乱数组：这会改变原数组
func Shuffle[T any](arr []T) {
	if len(arr) <= 0 {
		return
	}
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	r.Shuffle(len(arr), func(i, j int) {
		arr[i], arr[j] = arr[j], arr[i]
	})
}

// 打乱数组：Slower, use `copy` function
//
// 这不会改变原数组，而是返回打乱后的新数组
func ShuffleCopy[T any](arr []T) []T {
	if len(arr) <= 0 {
		return []T{}
	}
	target := make([]T, len(arr))
	copy(target, arr)
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	r.Shuffle(len(target), func(i, j int) {
		target[i], target[j] = target[j], target[i]
	})
	return target
}
