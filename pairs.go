package gchain

import "iter"

// Pair 把一个 key-value 元素表示为普通值。
type Pair[K any, V any] struct {
	Key   K
	Value V
}

// Pairs 表示 key-value 的 deferred sequence。
type Pairs[K comparable, V any] struct {
	seq  iter.Seq2[K, V]
	size sizeHint
}

// FromMap 从 Go map 创建 Pairs，不复制元素。
func FromMap[Map ~map[K]V, K comparable, V any](values Map) Pairs[K, V] {
	return Pairs[K, V]{
		size: exactSizeHint(len(values)),
		seq: func(yield func(K, V) bool) {
			for key, value := range values {
				if !yield(key, value) {
					return
				}
			}
		},
	}
}

// FromSeq2 从标准库 pair iterator 创建 Pairs。
func FromSeq2[K comparable, V any](seq iter.Seq2[K, V]) Pairs[K, V] {
	return Pairs[K, V]{seq: seq}
}

// Seq2 返回当前 Pairs 的标准库 pair iterator。
func (p Pairs[K, V]) Seq2() iter.Seq2[K, V] {
	if p.seq == nil {
		return func(func(K, V) bool) {}
	}
	return p.seq
}

// Filter deferred 地保留匹配 f 的 key-value pair。
func (p Pairs[K, V]) Filter(f func(K, V) bool) Pairs[K, V] {
	if f == nil {
		panic("gchain: nil Filter function")
	}

	return Pairs[K, V]{
		size: p.size.asUpperBound(),
		seq: func(yield func(K, V) bool) {
			p.Seq2()(func(key K, value V) bool {
				if !f(key, value) {
					return true
				}
				return yield(key, value)
			})
		},
	}
}

// FilterKeys deferred 地按 key 保留 key-value pair。
func (p Pairs[K, V]) FilterKeys(f func(K) bool) Pairs[K, V] {
	if f == nil {
		panic("gchain: nil FilterKeys function")
	}

	return Pairs[K, V]{
		size: p.size.asUpperBound(),
		seq: func(yield func(K, V) bool) {
			p.Seq2()(func(key K, value V) bool {
				if !f(key) {
					return true
				}
				return yield(key, value)
			})
		},
	}
}

// FilterValues deferred 地按 value 保留 key-value pair。
func (p Pairs[K, V]) FilterValues(f func(V) bool) Pairs[K, V] {
	if f == nil {
		panic("gchain: nil FilterValues function")
	}

	return Pairs[K, V]{
		size: p.size.asUpperBound(),
		seq: func(yield func(K, V) bool) {
			p.Seq2()(func(key K, value V) bool {
				if !f(value) {
					return true
				}
				return yield(key, value)
			})
		},
	}
}

// Map deferred 地把每个 key-value pair 映射为单值。
func (p Pairs[K, V]) Map[U any](f func(K, V) U) Chain[U] {
	if f == nil {
		panic("gchain: nil Map function")
	}

	return Chain[U]{
		size: p.size,
		seq: func(yield func(U) bool) {
			p.Seq2()(func(key K, value V) bool {
				return yield(f(key, value))
			})
		},
	}
}

// MapFilter deferred 地映射 key-value pair，并只保留 keep 为 true 的结果。
func (p Pairs[K, V]) MapFilter[U any](f func(K, V) (U, bool)) Chain[U] {
	if f == nil {
		panic("gchain: nil MapFilter function")
	}

	return Chain[U]{
		size: p.size.asUpperBound(),
		seq: func(yield func(U) bool) {
			p.Seq2()(func(key K, value V) bool {
				mapped, keep := f(key, value)
				if !keep {
					return true
				}
				return yield(mapped)
			})
		},
	}
}

// MapKeys deferred 地映射 key，并保留 value。
func (p Pairs[K, V]) MapKeys[K2 comparable](f func(K) K2) Pairs[K2, V] {
	if f == nil {
		panic("gchain: nil MapKeys function")
	}

	return Pairs[K2, V]{
		size: p.size,
		seq: func(yield func(K2, V) bool) {
			p.Seq2()(func(key K, value V) bool {
				return yield(f(key), value)
			})
		},
	}
}

// MapValues deferred 地映射 value，并保留 key。
func (p Pairs[K, V]) MapValues[U any](f func(V) U) Pairs[K, U] {
	if f == nil {
		panic("gchain: nil MapValues function")
	}

	return Pairs[K, U]{
		size: p.size,
		seq: func(yield func(K, U) bool) {
			p.Seq2()(func(key K, value V) bool {
				return yield(key, f(value))
			})
		},
	}
}

// Concat deferred 地把当前 pair sequence 和另一个 pair sequence 顺序拼接。
func (p Pairs[K, V]) Concat(other Pairs[K, V]) Pairs[K, V] {
	return Pairs[K, V]{
		size: p.size.concat(other.size),
		seq: func(yield func(K, V) bool) {
			keepGoing := true
			p.Seq2()(func(key K, value V) bool {
				if !yield(key, value) {
					keepGoing = false
					return false
				}
				return true
			})
			if !keepGoing {
				return
			}
			other.Seq2()(func(key K, value V) bool {
				return yield(key, value)
			})
		},
	}
}

// Take deferred 地保留最多 n 个 pair。
func (p Pairs[K, V]) Take(n int) Pairs[K, V] {
	if n < 0 {
		panic("gchain: negative Take count")
	}

	return Pairs[K, V]{
		size: p.size.take(n),
		seq: func(yield func(K, V) bool) {
			if n == 0 {
				return
			}

			count := 0
			p.Seq2()(func(key K, value V) bool {
				count++
				if !yield(key, value) {
					return false
				}
				return count < n
			})
		},
	}
}

// Drop deferred 地跳过前 n 个 pair。
func (p Pairs[K, V]) Drop(n int) Pairs[K, V] {
	if n < 0 {
		panic("gchain: negative Drop count")
	}

	return Pairs[K, V]{
		size: p.size.drop(n),
		seq: func(yield func(K, V) bool) {
			skipped := 0
			p.Seq2()(func(key K, value V) bool {
				if skipped < n {
					skipped++
					return true
				}
				return yield(key, value)
			})
		},
	}
}

// TakeWhile deferred 地保留前缀中连续匹配 f 的 pair，并在首次不匹配时停止消费。
func (p Pairs[K, V]) TakeWhile(f func(K, V) bool) Pairs[K, V] {
	if f == nil {
		panic("gchain: nil TakeWhile function")
	}

	return Pairs[K, V]{
		size: p.size.asUpperBound(),
		seq: func(yield func(K, V) bool) {
			p.Seq2()(func(key K, value V) bool {
				if !f(key, value) {
					return false
				}
				return yield(key, value)
			})
		},
	}
}

// DropWhile deferred 地跳过前缀中连续匹配 f 的 pair。
func (p Pairs[K, V]) DropWhile(f func(K, V) bool) Pairs[K, V] {
	if f == nil {
		panic("gchain: nil DropWhile function")
	}

	return Pairs[K, V]{
		size: p.size.asUpperBound(),
		seq: func(yield func(K, V) bool) {
			dropping := true
			p.Seq2()(func(key K, value V) bool {
				if dropping && f(key, value) {
					return true
				}
				dropping = false
				return yield(key, value)
			})
		},
	}
}

// Keys deferred 地把 pair sequence 转为 key sequence。
func (p Pairs[K, V]) Keys() Chain[K] {
	return Chain[K]{
		size: p.size,
		seq: func(yield func(K) bool) {
			p.Seq2()(func(key K, value V) bool {
				return yield(key)
			})
		},
	}
}

// Values deferred 地把 pair sequence 转为 value sequence。
func (p Pairs[K, V]) Values() Chain[V] {
	return Chain[V]{
		size: p.size,
		seq: func(yield func(V) bool) {
			p.Seq2()(func(key K, value V) bool {
				return yield(value)
			})
		},
	}
}

// Reverse deferred 地反转 pair sequence 的顺序。
//
// Reverse 会先收集全部 pair，再按相反顺序输出。
func (p Pairs[K, V]) Reverse() Pairs[K, V] {
	return Pairs[K, V]{
		size: p.size,
		seq: func(yield func(K, V) bool) {
			pairs := p.ToPairs()
			for i := len(pairs) - 1; i >= 0; i-- {
				pair := pairs[i]
				if !yield(pair.Key, pair.Value) {
					return
				}
			}
		},
	}
}

// ToMap 把 Pairs 消费为 Go map。
//
// 如果同一个 key 出现多次，最后一个 value 生效。
func (p Pairs[K, V]) ToMap() map[K]V {
	values := make(map[K]V, p.size.exactCapacity())
	p.Seq2()(func(key K, value V) bool {
		values[key] = value
		return true
	})
	return values
}

// ToPairs 把 Pairs 消费为 Pair slice。
func (p Pairs[K, V]) ToPairs() []Pair[K, V] {
	values := make([]Pair[K, V], 0, p.size.exactCapacity())
	p.Seq2()(func(key K, value V) bool {
		values = append(values, Pair[K, V]{
			Key:   key,
			Value: value,
		})
		return true
	})
	return values
}

// Count 消费 Pairs，并返回 pair 数量。
func (p Pairs[K, V]) Count() int {
	count := 0
	p.Seq2()(func(K, V) bool {
		count++
		return true
	})
	return count
}

// First 最多消费一个 pair。
func (p Pairs[K, V]) First() (K, V, bool) {
	var firstKey K
	var firstValue V
	ok := false
	p.Seq2()(func(key K, value V) bool {
		firstKey = key
		firstValue = value
		ok = true
		return false
	})
	return firstKey, firstValue, ok
}

// Find 返回第一个匹配 f 的 pair，并在首次匹配后停止。
func (p Pairs[K, V]) Find(f func(K, V) bool) (K, V, bool) {
	if f == nil {
		panic("gchain: nil Find function")
	}

	var foundKey K
	var foundValue V
	ok := false
	p.Seq2()(func(key K, value V) bool {
		if !f(key, value) {
			return true
		}
		foundKey = key
		foundValue = value
		ok = true
		return false
	})
	return foundKey, foundValue, ok
}

// Any 判断是否有任一 pair 匹配 f，并在首次匹配后停止。
func (p Pairs[K, V]) Any(f func(K, V) bool) bool {
	if f == nil {
		panic("gchain: nil Any function")
	}

	found := false
	p.Seq2()(func(key K, value V) bool {
		if !f(key, value) {
			return true
		}
		found = true
		return false
	})
	return found
}

// All 判断是否所有 pair 都匹配 f，并在首次不匹配后停止。
func (p Pairs[K, V]) All(f func(K, V) bool) bool {
	if f == nil {
		panic("gchain: nil All function")
	}

	all := true
	p.Seq2()(func(key K, value V) bool {
		if f(key, value) {
			return true
		}
		all = false
		return false
	})
	return all
}

// Reduce 消费 Pairs，并把每个 key-value pair 折叠进 accumulator。
func (p Pairs[K, V]) Reduce[U any](zero U, f func(U, K, V) U) U {
	if f == nil {
		panic("gchain: nil Reduce function")
	}

	acc := zero
	p.Seq2()(func(key K, value V) bool {
		acc = f(acc, key, value)
		return true
	})
	return acc
}

// ForEach 消费 Pairs，并对每个 key-value pair 调用 f。
func (p Pairs[K, V]) ForEach(f func(K, V)) {
	if f == nil {
		panic("gchain: nil ForEach function")
	}

	p.Seq2()(func(key K, value V) bool {
		f(key, value)
		return true
	})
}
