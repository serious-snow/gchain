// Package gchain 提供面向 Go 集合和 iterator 的 deferred 链式操作。
package gchain

import "iter"

// Chain 表示单值 deferred sequence。
//
// Chain 在 ToSlice 或 Count 这类 terminal operation 运行前不会消费 source。
type Chain[T any] struct {
	seq  iter.Seq[T]
	size sizeHint
}

// From 从 slice 创建 Chain，不复制元素。
//
// From 会捕获调用时的 slice header；terminal operation 前的元素修改会在消费时可见。
func From[Slice ~[]T, T any](values Slice) Chain[T] {
	return Chain[T]{
		size: exactSizeHint(len(values)),
		seq: func(yield func(T) bool) {
			for _, value := range values {
				if !yield(value) {
					return
				}
			}
		},
	}
}

// FromSeq 从标准库 iterator 创建 Chain。
func FromSeq[T any](seq iter.Seq[T]) Chain[T] {
	return Chain[T]{seq: seq}
}

// Seq 返回当前 Chain 的标准库 iterator。
func (c Chain[T]) Seq() iter.Seq[T] {
	if c.seq == nil {
		return func(func(T) bool) {}
	}
	return c.seq
}

// Map deferred 地把每个元素映射为另一个值。
func (c Chain[T]) Map[U any](f func(T) U) Chain[U] {
	if f == nil {
		panic("gchain: nil Map function")
	}

	return Chain[U]{
		size: c.size,
		seq: func(yield func(U) bool) {
			c.Seq()(func(value T) bool {
				return yield(f(value))
			})
		},
	}
}

// MapIndexed deferred 地把索引和值映射为另一个值。
func (c Chain[T]) MapIndexed[U any](f func(int, T) U) Chain[U] {
	if f == nil {
		panic("gchain: nil MapIndexed function")
	}

	return Chain[U]{
		size: c.size,
		seq: func(yield func(U) bool) {
			index := 0
			c.Seq()(func(value T) bool {
				current := index
				index++
				return yield(f(current, value))
			})
		},
	}
}

// MapFilter deferred 地映射元素，并只保留 keep 为 true 的结果。
func (c Chain[T]) MapFilter[U any](f func(T) (U, bool)) Chain[U] {
	if f == nil {
		panic("gchain: nil MapFilter function")
	}

	return Chain[U]{
		size: c.size.asUpperBound(),
		seq: func(yield func(U) bool) {
			c.Seq()(func(value T) bool {
				mapped, keep := f(value)
				if !keep {
					return true
				}
				return yield(mapped)
			})
		},
	}
}

// MapFilterIndexed deferred 地按索引和值映射元素，并只保留 keep 为 true 的结果。
func (c Chain[T]) MapFilterIndexed[U any](f func(int, T) (U, bool)) Chain[U] {
	if f == nil {
		panic("gchain: nil MapFilterIndexed function")
	}

	return Chain[U]{
		size: c.size.asUpperBound(),
		seq: func(yield func(U) bool) {
			index := 0
			c.Seq()(func(value T) bool {
				current := index
				index++
				mapped, keep := f(current, value)
				if !keep {
					return true
				}
				return yield(mapped)
			})
		},
	}
}

// Concat deferred 地把当前 sequence 和另一个 sequence 顺序拼接。
func (c Chain[T]) Concat(other Chain[T]) Chain[T] {
	return Chain[T]{
		size: c.size.concat(other.size),
		seq: func(yield func(T) bool) {
			keepGoing := true
			c.Seq()(func(value T) bool {
				if !yield(value) {
					keepGoing = false
					return false
				}
				return true
			})
			if !keepGoing {
				return
			}
			other.Seq()(func(value T) bool {
				return yield(value)
			})
		},
	}
}

// Filter deferred 地保留匹配 f 的元素。
func (c Chain[T]) Filter(f func(T) bool) Chain[T] {
	if f == nil {
		panic("gchain: nil Filter function")
	}

	return Chain[T]{
		size: c.size.asUpperBound(),
		seq: func(yield func(T) bool) {
			c.Seq()(func(value T) bool {
				if !f(value) {
					return true
				}
				return yield(value)
			})
		},
	}
}

// FilterIndexed deferred 地按索引和值保留元素。
func (c Chain[T]) FilterIndexed(f func(int, T) bool) Chain[T] {
	if f == nil {
		panic("gchain: nil FilterIndexed function")
	}

	return Chain[T]{
		size: c.size.asUpperBound(),
		seq: func(yield func(T) bool) {
			index := 0
			c.Seq()(func(value T) bool {
				current := index
				index++
				if !f(current, value) {
					return true
				}
				return yield(value)
			})
		},
	}
}

// Take deferred 地保留最多 n 个元素。
func (c Chain[T]) Take(n int) Chain[T] {
	if n < 0 {
		panic("gchain: negative Take count")
	}

	return Chain[T]{
		size: c.size.take(n),
		seq: func(yield func(T) bool) {
			if n == 0 {
				return
			}

			count := 0
			c.Seq()(func(value T) bool {
				count++
				if !yield(value) {
					return false
				}
				return count < n
			})
		},
	}
}

// Drop deferred 地跳过前 n 个元素。
func (c Chain[T]) Drop(n int) Chain[T] {
	if n < 0 {
		panic("gchain: negative Drop count")
	}

	return Chain[T]{
		size: c.size.drop(n),
		seq: func(yield func(T) bool) {
			skipped := 0
			c.Seq()(func(value T) bool {
				if skipped < n {
					skipped++
					return true
				}
				return yield(value)
			})
		},
	}
}

// TakeWhile deferred 地保留前缀中连续匹配 f 的元素，并在首次不匹配时停止消费。
func (c Chain[T]) TakeWhile(f func(T) bool) Chain[T] {
	if f == nil {
		panic("gchain: nil TakeWhile function")
	}

	return Chain[T]{
		size: c.size.asUpperBound(),
		seq: func(yield func(T) bool) {
			c.Seq()(func(value T) bool {
				if !f(value) {
					return false
				}
				return yield(value)
			})
		},
	}
}

// DropWhile deferred 地跳过前缀中连续匹配 f 的元素。
func (c Chain[T]) DropWhile(f func(T) bool) Chain[T] {
	if f == nil {
		panic("gchain: nil DropWhile function")
	}

	return Chain[T]{
		size: c.size.asUpperBound(),
		seq: func(yield func(T) bool) {
			dropping := true
			c.Seq()(func(value T) bool {
				if dropping && f(value) {
					return true
				}
				dropping = false
				return yield(value)
			})
		},
	}
}

// Chunk deferred 地按 n 个元素分块，并保留不足 n 的尾块。
func (c Chain[T]) Chunk(n int) Chunks[T] {
	if n <= 0 {
		panic("gchain: non-positive Chunk size")
	}

	return Chunks[T]{
		size: c.size.chunk(n),
		seq: func(yield func([]T) bool) {
			chunk := make([]T, 0, n)
			c.Seq()(func(value T) bool {
				chunk = append(chunk, value)
				if len(chunk) < n {
					return true
				}

				current := chunk
				chunk = make([]T, 0, n)
				return yield(current)
			})
			if len(chunk) > 0 {
				yield(chunk)
			}
		},
	}
}

// FlatMap deferred 地把每个元素展开为多个元素。
func (c Chain[T]) FlatMap[U any](f func(T) []U) Chain[U] {
	if f == nil {
		panic("gchain: nil FlatMap function")
	}

	return Chain[U]{
		seq: func(yield func(U) bool) {
			c.Seq()(func(value T) bool {
				for _, mapped := range f(value) {
					if !yield(mapped) {
						return false
					}
				}
				return true
			})
		},
	}
}

// DistinctBy deferred 地保留第一次出现的 key。
//
// DistinctBy 会记住已经见过的 key，因此会保留部分状态，但不会 materialize 全量 source。
func (c Chain[T]) DistinctBy[K comparable](key func(T) K) Chain[T] {
	if key == nil {
		panic("gchain: nil DistinctBy function")
	}

	return Chain[T]{
		size: c.size.asUpperBound(),
		seq: func(yield func(T) bool) {
			seen := make(map[K]struct{})
			c.Seq()(func(value T) bool {
				k := key(value)
				if _, ok := seen[k]; ok {
					return true
				}
				seen[k] = struct{}{}
				return yield(value)
			})
		},
	}
}

// Reverse deferred 地反转元素顺序。
//
// Reverse 会先收集全部元素，再按相反顺序输出。
func (c Chain[T]) Reverse() Chain[T] {
	return Chain[T]{
		size: c.size,
		seq: func(yield func(T) bool) {
			values := c.ToSlice()
			for i := len(values) - 1; i >= 0; i-- {
				if !yield(values[i]) {
					return
				}
			}
		},
	}
}

// Enumerate deferred 地把每个元素和从 0 开始的索引组成 pair sequence。
func (c Chain[T]) Enumerate() Pairs[int, T] {
	return Pairs[int, T]{
		size: c.size,
		seq: func(yield func(int, T) bool) {
			index := 0
			c.Seq()(func(value T) bool {
				current := index
				index++
				return yield(current, value)
			})
		},
	}
}

// GroupBy 在 terminal consumption 期间按 key 聚合元素。
//
// GroupBy 是 deferred operation，也是 buffering operation：消费时读取上游一次，
// 并保留分组结果，然后再产出完整分组。
func (c Chain[T]) GroupBy[K comparable](key func(T) K) Groups[K, T] {
	if key == nil {
		panic("gchain: nil GroupBy function")
	}

	return Groups[K, T]{
		size: c.size.asUpperBound(),
		seq: func(yield func(K, []T) bool) {
			groups := make(map[K][]T)
			c.Seq()(func(value T) bool {
				groupKey := key(value)
				groups[groupKey] = append(groups[groupKey], value)
				return true
			})

			for groupKey, values := range groups {
				if !yield(groupKey, values) {
					return
				}
			}
		},
	}
}

// ToSlice 消费 Chain，并返回包含全部元素的 slice。
func (c Chain[T]) ToSlice() []T {
	values := make([]T, 0, c.size.exactCapacity())
	c.Seq()(func(value T) bool {
		values = append(values, value)
		return true
	})
	return values
}

// ToMap 使用 f 产出 key-value pair，并把 Chain 消费为 Go map。
//
// 如果 f 多次返回同一个 key，最后一个 value 生效。
func (c Chain[T]) ToMap[K comparable, V any](f func(T) (K, V)) map[K]V {
	if f == nil {
		panic("gchain: nil ToMap function")
	}

	values := make(map[K]V, c.size.exactCapacity())
	c.Seq()(func(value T) bool {
		key, mapped := f(value)
		values[key] = mapped
		return true
	})
	return values
}

// ToMapBy 使用 f 产出 key，并把原元素作为 value 消费为 Go map。
//
// 如果 f 多次返回同一个 key，最后一个元素生效。
func (c Chain[T]) ToMapBy[K comparable](f func(T) K) map[K]T {
	if f == nil {
		panic("gchain: nil ToMapBy function")
	}

	values := make(map[K]T, c.size.exactCapacity())
	c.Seq()(func(value T) bool {
		values[f(value)] = value
		return true
	})
	return values
}

// ToGroups 使用 f 对元素分组，并把 Chain 消费为 map[K][]T。
func (c Chain[T]) ToGroups[K comparable](f func(T) K) map[K][]T {
	if f == nil {
		panic("gchain: nil ToGroups function")
	}

	groups := make(map[K][]T)
	c.Seq()(func(value T) bool {
		key := f(value)
		groups[key] = append(groups[key], value)
		return true
	})
	return groups
}

// Count 消费 Chain，并返回元素数量。
func (c Chain[T]) Count() int {
	count := 0
	c.Seq()(func(T) bool {
		count++
		return true
	})
	return count
}

// CountBy 按 key 统计元素数量。
func (c Chain[T]) CountBy[K comparable](f func(T) K) map[K]int {
	if f == nil {
		panic("gchain: nil CountBy function")
	}

	counts := make(map[K]int)
	c.Seq()(func(value T) bool {
		counts[f(value)]++
		return true
	})
	return counts
}

// First 最多消费一个元素。
func (c Chain[T]) First() (T, bool) {
	var first T
	ok := false
	c.Seq()(func(value T) bool {
		first = value
		ok = true
		return false
	})
	return first, ok
}

// Find 返回第一个匹配 f 的元素，并在首次匹配后停止。
func (c Chain[T]) Find(f func(T) bool) (T, bool) {
	if f == nil {
		panic("gchain: nil Find function")
	}

	var found T
	ok := false
	c.Seq()(func(value T) bool {
		if !f(value) {
			return true
		}
		found = value
		ok = true
		return false
	})
	return found, ok
}

// Any 判断是否有任一元素匹配 f，并在首次匹配后停止。
func (c Chain[T]) Any(f func(T) bool) bool {
	if f == nil {
		panic("gchain: nil Any function")
	}

	found := false
	c.Seq()(func(value T) bool {
		if !f(value) {
			return true
		}
		found = true
		return false
	})
	return found
}

// All 判断是否所有元素都匹配 f，并在首次不匹配后停止。
func (c Chain[T]) All(f func(T) bool) bool {
	if f == nil {
		panic("gchain: nil All function")
	}

	all := true
	c.Seq()(func(value T) bool {
		if f(value) {
			return true
		}
		all = false
		return false
	})
	return all
}

// Reduce 消费 Chain，并把每个元素折叠进 accumulator。
func (c Chain[T]) Reduce[U any](zero U, f func(U, T) U) U {
	if f == nil {
		panic("gchain: nil Reduce function")
	}

	acc := zero
	c.Seq()(func(value T) bool {
		acc = f(acc, value)
		return true
	})
	return acc
}

// ForEach 消费 Chain，并对每个元素调用 f。
func (c Chain[T]) ForEach(f func(T)) {
	if f == nil {
		panic("gchain: nil ForEach function")
	}

	c.Seq()(func(value T) bool {
		f(value)
		return true
	})
}
