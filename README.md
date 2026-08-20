# GChain

GChain 是一个 Go 1.27+ 的链式集合库，面向 slice、Go map 和标准库 iterator。

核心目标：

- 调用链时不执行，直到 terminal operation 才消费 source
- 普通 `Map` / `Filter` / `Take` / `Drop` 不 materialize 中间 slice
- `GroupBy` 这类操作仍然 deferred，但执行时会 buffering
- 底层使用标准库 `iter.Seq` / `iter.Seq2`

## Installation

```sh
go get github.com/serious-snow/gchain@v0.0.1
```

```go
import "github.com/serious-snow/gchain"
```

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

## Method Reference

### Sources

| Function | Returns | Description |
| --- | --- | --- |
| `From(values)` | `Chain[T]` | 从 slice 创建 chain，不复制元素。 |
| `FromSeq(seq)` | `Chain[T]` | 从 `iter.Seq[T]` 创建 chain。 |
| `FromMap(values)` | `Pairs[K,V]` | 从 Go map 创建 key-value chain，不复制元素。 |
| `FromSeq2(seq)` | `Pairs[K,V]` | 从 `iter.Seq2[K,V]` 创建 key-value chain。 |

### Chain[T]

| Method | Returns | Description | Execution |
| --- | --- | --- | --- |
| `Map(func(T) U)` | `Chain[U]` | 映射元素类型。 | deferred, streaming |
| `Filter(func(T) bool)` | `Chain[T]` | 保留匹配元素。 | deferred, streaming |
| `Take(n)` | `Chain[T]` | 最多保留前 `n` 个元素，支持早停。 | deferred, streaming |
| `Drop(n)` | `Chain[T]` | 跳过前 `n` 个元素。 | deferred, streaming |
| `FlatMap(func(T) []U)` | `Chain[U]` | 把一个元素展开为多个元素。 | deferred, streaming |
| `DistinctBy(func(T) K)` | `Chain[T]` | 按 key 去重，保留首次出现。 | deferred, stateful |
| `Reverse()` | `Chain[T]` | 反转元素顺序。 | deferred, buffering |
| `Enumerate()` | `Pairs[int,T]` | 给元素附加从 0 开始的索引。 | deferred, streaming |
| `GroupBy(func(T) K)` | `Groups[K,T]` | 按 key 分组为 `[]T`。 | deferred, buffering |
| `ToSlice()` | `[]T` | 收集为 slice。 | terminal |
| `ToMap(func(T) (K,V))` | `map[K]V` | 用函数产出 key-value，重复 key 时 last wins。 | terminal |
| `ToMapBy(func(T) K)` | `map[K]T` | 用函数产出 key，原元素作为 value。 | terminal |
| `ToGroups(func(T) K)` | `map[K][]T` | 直接收集为分组 map。 | terminal |
| `Count()` | `int` | 统计元素数量。 | terminal |
| `First()` | `(T, bool)` | 返回第一个元素，空 chain 返回 `false`。 | terminal, early-stop |
| `Any(func(T) bool)` | `bool` | 任一元素匹配则返回 `true`。 | terminal, early-stop |
| `All(func(T) bool)` | `bool` | 所有元素匹配才返回 `true`。 | terminal, early-stop |
| `Reduce(zero, func(U,T) U)` | `U` | 把元素折叠进 accumulator。 | terminal |
| `ForEach(func(T))` | 无 | 对每个元素执行函数。 | terminal |
| `Seq()` | `iter.Seq[T]` | 转回标准库 iterator。 | adapter |

### Groups[K,T]

| Method | Returns | Description | Execution |
| --- | --- | --- | --- |
| `Filter(func(K, []T) bool)` | `Groups[K,T]` | 保留匹配的 group。 | deferred, streaming over groups |
| `Map(func(K, []T) U)` | `Chain[U]` | 把 group 映射为单值。 | deferred, streaming over groups |
| `MapValues(func([]T) U)` | `Pairs[K,U]` | 映射 group value，并保留 key。 | deferred, streaming over groups |
| `Keys()` | `Chain[K]` | 提取 group key。 | deferred, streaming over groups |
| `ToMap()` | `map[K][]T` | 收集为分组 map。 | terminal |
| `ToPairs()` | `[]Pair[K,[]T]` | 收集为 pair slice。 | terminal |
| `Count()` | `int` | 统计 group 数量。 | terminal |
| `ForEach(func(K, []T))` | 无 | 对每个 group 执行函数。 | terminal |
| `Seq2()` | `iter.Seq2[K,[]T]` | 转回标准库 pair iterator。 | adapter |

### Pairs[K,V]

| Method | Returns | Description | Execution |
| --- | --- | --- | --- |
| `Filter(func(K,V) bool)` | `Pairs[K,V]` | 保留匹配的 key-value pair。 | deferred, streaming |
| `Map(func(K,V) U)` | `Chain[U]` | 把 pair 映射为单值。 | deferred, streaming |
| `MapKeys(func(K) K2)` | `Pairs[K2,V]` | 映射 key，并保留 value。 | deferred, streaming |
| `MapValues(func(V) U)` | `Pairs[K,U]` | 映射 value，并保留 key。 | deferred, streaming |
| `Reverse()` | `Pairs[K,V]` | 反转 pair 顺序。 | deferred, buffering |
| `Keys()` | `Chain[K]` | 提取所有 key。 | deferred, streaming |
| `Values()` | `Chain[V]` | 提取所有 value。 | deferred, streaming |
| `ToMap()` | `map[K]V` | 收集为 Go map，重复 key 时 last wins。 | terminal |
| `ToPairs()` | `[]Pair[K,V]` | 收集为 pair slice。 | terminal |
| `Count()` | `int` | 统计 pair 数量。 | terminal |
| `ForEach(func(K,V))` | 无 | 对每个 pair 执行函数。 | terminal |
| `Seq2()` | `iter.Seq2[K,V]` | 转回标准库 pair iterator。 | adapter |

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
