package gchain

import "iter"

// Pair 把一个 key-value 元素表示为普通值。
type Pair[K any, V any] struct {
	Key   K
	Value V
}

// Pairs 表示 key-value 的 deferred sequence。
type Pairs[K comparable, V any] struct {
	seq iter.Seq2[K, V]
}

// FromMap 从 Go map 创建 Pairs，不复制元素。
func FromMap[Map ~map[K]V, K comparable, V any](values Map) Pairs[K, V] {
	return Pairs[K, V]{
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

// Map deferred 地把每个 key-value pair 映射为单值。
func (p Pairs[K, V]) Map[U any](f func(K, V) U) Chain[U] {
	if f == nil {
		panic("gchain: nil Map function")
	}

	return Chain[U]{
		seq: func(yield func(U) bool) {
			p.Seq2()(func(key K, value V) bool {
				return yield(f(key, value))
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
		seq: func(yield func(K, U) bool) {
			p.Seq2()(func(key K, value V) bool {
				return yield(key, f(value))
			})
		},
	}
}

// Keys deferred 地把 pair sequence 转为 key sequence。
func (p Pairs[K, V]) Keys() Chain[K] {
	return Chain[K]{
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
	values := make(map[K]V)
	p.Seq2()(func(key K, value V) bool {
		values[key] = value
		return true
	})
	return values
}

// ToPairs 把 Pairs 消费为 Pair slice。
func (p Pairs[K, V]) ToPairs() []Pair[K, V] {
	values := make([]Pair[K, V], 0)
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
