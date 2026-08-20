# GChain

GChain 是一个 Go 1.27+ 的链式集合库，面向 slice、Go map 和标准库 iterator。

核心目标：

- 调用链时不执行，直到 terminal operation 才消费 source
- 普通 `Map` / `Filter` / `Take` / `Drop` 不 materialize 中间 slice
- `GroupBy` 这类操作仍然 deferred，但执行时会 buffering
- 底层使用标准库 `iter.Seq` / `iter.Seq2`

## Example

```go
namesByTeam := gchain.From(users).
	Filter(func(user User) bool {
		return user.Active
	}).
	GroupBy(func(user User) string {
		return user.Team
	}).
	MapValues(func(users []User) []string {
		return gchain.From(users).
			Map(func(user User) string {
				return user.Name
			}).
			ToSlice()
	}).
	ToMap()
```

```go
ports := gchain.FromMap(config).
	Filter(func(name string, port int) bool {
		return port > 0
	}).
	MapKeys(strings.ToUpper).
	ToMap()
```

## Operation Semantics

**Deferred Operation** 记录工作，但不会立刻执行；只有 terminal operation 消费 chain 时才会运行。

**Streaming Operation** 可以逐个元素向下游传递，不保留全部上游元素。

**Buffering Operation** 仍然等到 terminal consumption 才执行，但执行时会保留它需要的中间状态，并只消费上游一次。`GroupBy` 是 buffering operation，因为它必须先构建完整分组，再产出 group。

## API Shape

Single-value sequences use `Chain[T]`:

```go
gchain.From(values)
gchain.FromSeq(seq)

chain.Map(func(T) U)
chain.Filter(func(T) bool)
chain.Take(n)
chain.Drop(n)
chain.FlatMap(func(T) []U)
chain.DistinctBy(func(T) K)
chain.Reverse()
chain.Enumerate()
chain.GroupBy(func(T) K) // Groups[K,T]

chain.ToSlice()
chain.ToMap(func(T) (K, V))
chain.ToMapBy(func(T) K)
chain.ToGroups(func(T) K)
chain.Count()
chain.First()
chain.Any(func(T) bool)
chain.All(func(T) bool)
chain.Reduce(zero, func(U, T) U)
chain.ForEach(func(T))
chain.Seq()
```

Grouping results use `Groups[K,T]`:

```go
groups.Filter(func(K, []T) bool)
groups.Map(func(K, []T) U)
groups.MapValues(func([]T) U) // Pairs[K,U]
groups.Keys()

groups.ToMap()
groups.ToPairs()
groups.Count()
groups.ForEach(func(K, []T))
groups.Seq2()
```

Key-value sequences use `Pairs[K,V]`:

```go
gchain.FromMap(values)
gchain.FromSeq2(seq)

pairs.Filter(func(K, V) bool)
pairs.Map(func(K, V) U)
pairs.MapKeys(func(K) K2)
pairs.MapValues(func(V) U)
pairs.Reverse()
pairs.Keys()
pairs.Values()

pairs.ToMap()
pairs.ToPairs()
pairs.Count()
pairs.ForEach(func(K, V))
pairs.Seq2()
```

## Notes

- Duplicate keys use last-wins semantics in `ToMap` and `Pairs.ToMap`.
- `DistinctBy` keeps first occurrence for each key.
- `Reverse` buffers the full source before yielding.
- Go map iteration order is not stable.
- `From` does not copy the slice; slice element changes before terminal consumption are visible.
- `FromMap` does not copy the map; map changes before terminal consumption are visible.
- `FromSeq` and `FromSeq2` follow source semantics, including single-use iterators.
- Nil slice and nil map sources produce empty results.
- Nil operation functions panic.

## Not In The First Version

- Error-aware chains
- Stable ordering for Go map based operations
- Sorting and zip over iterators
- Concurrency safety beyond what the source itself provides
