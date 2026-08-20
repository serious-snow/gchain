package gchain

import "iter"

// Groups 表示 GroupBy 产生的 deferred grouping sequence。
//
// Groups 用独立类型承载分组结果，避免 Chain 和 Pairs 的泛型方法形成无限实例化环。
type Groups[K comparable, T any] struct {
	seq  iter.Seq2[K, []T]
	size sizeHint
}

// Seq2 返回当前 Groups 的标准库 pair iterator。
func (g Groups[K, T]) Seq2() iter.Seq2[K, []T] {
	if g.seq == nil {
		return func(func(K, []T) bool) {}
	}
	return g.seq
}

// Filter deferred 地保留匹配 f 的 group。
func (g Groups[K, T]) Filter(f func(K, []T) bool) Groups[K, T] {
	if f == nil {
		panic("gchain: nil Filter function")
	}

	return Groups[K, T]{
		size: g.size.asUpperBound(),
		seq: func(yield func(K, []T) bool) {
			g.Seq2()(func(key K, values []T) bool {
				if !f(key, values) {
					return true
				}
				return yield(key, values)
			})
		},
	}
}

// Map deferred 地把每个 group 映射为单值。
func (g Groups[K, T]) Map[U any](f func(K, []T) U) Chain[U] {
	if f == nil {
		panic("gchain: nil Map function")
	}

	return Chain[U]{
		size: g.size,
		seq: func(yield func(U) bool) {
			g.Seq2()(func(key K, values []T) bool {
				return yield(f(key, values))
			})
		},
	}
}

// MapValues deferred 地映射 group value，并回到 Pairs。
func (g Groups[K, T]) MapValues[U any](f func([]T) U) Pairs[K, U] {
	if f == nil {
		panic("gchain: nil MapValues function")
	}

	return Pairs[K, U]{
		size: g.size,
		seq: func(yield func(K, U) bool) {
			g.Seq2()(func(key K, values []T) bool {
				return yield(key, f(values))
			})
		},
	}
}

// Keys deferred 地把 grouping sequence 转为 key sequence。
func (g Groups[K, T]) Keys() Chain[K] {
	return Chain[K]{
		size: g.size,
		seq: func(yield func(K) bool) {
			g.Seq2()(func(key K, values []T) bool {
				return yield(key)
			})
		},
	}
}

// ToMap 把 Groups 消费为 map[K][]T。
func (g Groups[K, T]) ToMap() map[K][]T {
	values := make(map[K][]T, g.size.exactCapacity())
	g.Seq2()(func(key K, group []T) bool {
		values[key] = group
		return true
	})
	return values
}

// ToPairs 把 Groups 消费为 Pair slice。
func (g Groups[K, T]) ToPairs() []Pair[K, []T] {
	values := make([]Pair[K, []T], 0, g.size.exactCapacity())
	g.Seq2()(func(key K, group []T) bool {
		values = append(values, Pair[K, []T]{
			Key:   key,
			Value: group,
		})
		return true
	})
	return values
}

// Count 消费 Groups，并返回 group 数量。
func (g Groups[K, T]) Count() int {
	count := 0
	g.Seq2()(func(K, []T) bool {
		count++
		return true
	})
	return count
}

// ForEach 消费 Groups，并对每个 group 调用 f。
func (g Groups[K, T]) ForEach(f func(K, []T)) {
	if f == nil {
		panic("gchain: nil ForEach function")
	}

	g.Seq2()(func(key K, values []T) bool {
		f(key, values)
		return true
	})
}
