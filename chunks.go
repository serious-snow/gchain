package gchain

import "iter"

// Chunks 表示 Chunk 产生的 deferred chunk sequence。
//
// Chunks 用独立类型承载 []T sequence，避免 Chain[T] 的方法递归实例化 Chain[[]T]。
type Chunks[T any] struct {
	seq  iter.Seq[[]T]
	size sizeHint
}

// Seq 返回当前 Chunks 的标准库 iterator。
func (c Chunks[T]) Seq() iter.Seq[[]T] {
	if c.seq == nil {
		return func(func([]T) bool) {}
	}
	return c.seq
}

// Filter deferred 地保留匹配 f 的 chunk。
func (c Chunks[T]) Filter(f func([]T) bool) Chunks[T] {
	if f == nil {
		panic("gchain: nil Filter function")
	}

	return Chunks[T]{
		size: c.size.asUpperBound(),
		seq: func(yield func([]T) bool) {
			c.Seq()(func(chunk []T) bool {
				if !f(chunk) {
					return true
				}
				return yield(chunk)
			})
		},
	}
}

// Map deferred 地把每个 chunk 映射为单值。
func (c Chunks[T]) Map[U any](f func([]T) U) Chain[U] {
	if f == nil {
		panic("gchain: nil Map function")
	}

	return Chain[U]{
		size: c.size,
		seq: func(yield func(U) bool) {
			c.Seq()(func(chunk []T) bool {
				return yield(f(chunk))
			})
		},
	}
}

// Take deferred 地保留最多 n 个 chunk。
func (c Chunks[T]) Take(n int) Chunks[T] {
	if n < 0 {
		panic("gchain: negative Take count")
	}

	return Chunks[T]{
		size: c.size.take(n),
		seq: func(yield func([]T) bool) {
			if n == 0 {
				return
			}

			count := 0
			c.Seq()(func(chunk []T) bool {
				count++
				if !yield(chunk) {
					return false
				}
				return count < n
			})
		},
	}
}

// Drop deferred 地跳过前 n 个 chunk。
func (c Chunks[T]) Drop(n int) Chunks[T] {
	if n < 0 {
		panic("gchain: negative Drop count")
	}

	return Chunks[T]{
		size: c.size.drop(n),
		seq: func(yield func([]T) bool) {
			skipped := 0
			c.Seq()(func(chunk []T) bool {
				if skipped < n {
					skipped++
					return true
				}
				return yield(chunk)
			})
		},
	}
}

// ToSlice 消费 Chunks，并返回包含全部 chunk 的 slice。
func (c Chunks[T]) ToSlice() [][]T {
	values := make([][]T, 0, c.size.exactCapacity())
	c.Seq()(func(chunk []T) bool {
		values = append(values, chunk)
		return true
	})
	return values
}

// Count 消费 Chunks，并返回 chunk 数量。
func (c Chunks[T]) Count() int {
	count := 0
	c.Seq()(func([]T) bool {
		count++
		return true
	})
	return count
}

// First 最多消费一个 chunk。
func (c Chunks[T]) First() ([]T, bool) {
	var first []T
	ok := false
	c.Seq()(func(chunk []T) bool {
		first = chunk
		ok = true
		return false
	})
	return first, ok
}

// ForEach 消费 Chunks，并对每个 chunk 调用 f。
func (c Chunks[T]) ForEach(f func([]T)) {
	if f == nil {
		panic("gchain: nil ForEach function")
	}

	c.Seq()(func(chunk []T) bool {
		f(chunk)
		return true
	})
}
