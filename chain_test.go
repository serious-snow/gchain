package gchain_test

import (
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/serious-snow/gchain"
)

type testUser struct {
	ID     string
	Name   string
	Team   string
	Active bool
}

func TestMapFilterDeferred(t *testing.T) {
	calls := 0

	chain := gchain.From([]int{1, 2, 3}).
		Map(func(value int) int {
			calls++
			return value * 2
		}).
		Filter(func(value int) bool {
			calls++
			return value > 2
		})

	if calls != 0 {
		t.Fatalf("deferred operations ran before terminal operation: calls=%d", calls)
	}

	got := chain.ToSlice()
	want := []int{4, 6}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("ToSlice()=%v, want %v", got, want)
	}
	if calls != 6 {
		t.Fatalf("calls=%d, want 6", calls)
	}
}

func TestMapFilterOperations(t *testing.T) {
	t.Run("MapFilter", func(t *testing.T) {
		calls := 0

		chain := gchain.From([]int{1, 2, 3, 4}).
			MapFilter(func(value int) (string, bool) {
				calls++
				keep := value%2 == 0
				return strings.Repeat("x", value/2), keep
			})

		if calls != 0 {
			t.Fatalf("MapFilter ran before terminal operation: calls=%d", calls)
		}

		got := chain.ToSlice()
		want := []string{"x", "xx"}

		if !reflect.DeepEqual(got, want) {
			t.Fatalf("MapFilter().ToSlice()=%v, want %v", got, want)
		}
		if calls != 4 {
			t.Fatalf("MapFilter calls=%d, want 4", calls)
		}
	})

	t.Run("MapFilterIndexed", func(t *testing.T) {
		got := gchain.From([]string{"a", "b", "c", "d"}).
			MapFilterIndexed(func(index int, value string) (string, bool) {
				keep := index%2 == 1
				return strings.Repeat(value, index+1), keep
			}).
			ToSlice()

		if !reflect.DeepEqual(got, []string{"bb", "dddd"}) {
			t.Fatalf("MapFilterIndexed().ToSlice()=%v, want [bb dddd]", got)
		}
	})

	t.Run("StopsAfterDownstreamStop", func(t *testing.T) {
		pulled := 0
		calls := 0

		got := gchain.FromSeq(func(yield func(int) bool) {
			for _, value := range []int{1, 2, 3, 4} {
				pulled++
				if !yield(value) {
					return
				}
			}
		}).
			MapFilter(func(value int) (int, bool) {
				calls++
				keep := value%2 == 0
				return value * 10, keep
			}).
			Take(1).
			ToSlice()

		if !reflect.DeepEqual(got, []int{20}) {
			t.Fatalf("MapFilter().Take(1).ToSlice()=%v, want [20]", got)
		}
		if pulled != 2 || calls != 2 {
			t.Fatalf("pulled=%d calls=%d, want 2 and 2", pulled, calls)
		}
	})
}

func TestConcatOperation(t *testing.T) {
	t.Run("ToSlice", func(t *testing.T) {
		got := gchain.From([]int{1, 2}).
			Concat(gchain.From([]int{3, 4})).
			ToSlice()

		if !reflect.DeepEqual(got, []int{1, 2, 3, 4}) {
			t.Fatalf("Concat().ToSlice()=%v, want [1 2 3 4]", got)
		}
	})

	t.Run("DeferredAndEarlyStop", func(t *testing.T) {
		leftPulled := 0
		rightPulled := 0

		chain := gchain.FromSeq(func(yield func(int) bool) {
			for _, value := range []int{1, 2, 3} {
				leftPulled++
				if !yield(value) {
					return
				}
			}
		}).Concat(gchain.FromSeq(func(yield func(int) bool) {
			for _, value := range []int{4, 5} {
				rightPulled++
				if !yield(value) {
					return
				}
			}
		}))

		if leftPulled != 0 || rightPulled != 0 {
			t.Fatalf("Concat ran before terminal operation: left=%d right=%d", leftPulled, rightPulled)
		}

		got := chain.Take(2).ToSlice()
		if !reflect.DeepEqual(got, []int{1, 2}) {
			t.Fatalf("Concat().Take(2).ToSlice()=%v, want [1 2]", got)
		}
		if leftPulled != 2 || rightPulled != 0 {
			t.Fatalf("leftPulled=%d rightPulled=%d, want 2 and 0", leftPulled, rightPulled)
		}
	})
}

func TestIndexedOperations(t *testing.T) {
	mapped := gchain.From([]string{"a", "b", "c"}).
		MapIndexed(func(index int, value string) string {
			return strings.Repeat(value, index+1)
		}).
		ToSlice()
	if !reflect.DeepEqual(mapped, []string{"a", "bb", "ccc"}) {
		t.Fatalf("MapIndexed().ToSlice()=%v, want [a bb ccc]", mapped)
	}

	filtered := gchain.From([]string{"a", "b", "c", "d"}).
		FilterIndexed(func(index int, value string) bool {
			return index%2 == 0 || value == "d"
		}).
		ToSlice()
	if !reflect.DeepEqual(filtered, []string{"a", "c", "d"}) {
		t.Fatalf("FilterIndexed().ToSlice()=%v, want [a c d]", filtered)
	}
}

// nolint
func TestEarlyStopOperations(t *testing.T) {
	t.Run("First", func(t *testing.T) {
		pulled := 0

		first, ok := gchain.FromSeq(func(yield func(int) bool) {
			for _, value := range []int{1, 2, 3} {
				pulled++
				if !yield(value) {
					return
				}
			}
		}).First()

		if first != 1 || !ok {
			t.Fatalf("First()=(%d, %v), want (1, true)", first, ok)
		}
		if pulled != 1 {
			t.Fatalf("pulled=%d, want 1", pulled)
		}
	})

	t.Run("Any", func(t *testing.T) {
		pulled := 0

		got := gchain.FromSeq(func(yield func(int) bool) {
			for _, value := range []int{1, 2, 3, 4} {
				pulled++
				if !yield(value) {
					return
				}
			}
		}).Any(func(value int) bool {
			return value == 3
		})

		if !got {
			t.Fatal("Any()=false, want true")
		}
		if pulled != 3 {
			t.Fatalf("pulled=%d, want 3", pulled)
		}
	})

	t.Run("Find", func(t *testing.T) {
		pulled := 0

		found, ok := gchain.FromSeq(func(yield func(int) bool) {
			for _, value := range []int{1, 2, 3, 4} {
				pulled++
				if !yield(value) {
					return
				}
			}
		}).Find(func(value int) bool {
			return value == 3
		})

		if found != 3 || !ok {
			t.Fatalf("Find()=(%d, %v), want (3, true)", found, ok)
		}
		if pulled != 3 {
			t.Fatalf("pulled=%d, want 3", pulled)
		}
	})

	t.Run("FindNone", func(t *testing.T) {
		found, ok := gchain.From([]int{1, 2}).Find(func(value int) bool {
			return value == 3
		})

		if found != 0 || ok {
			t.Fatalf("Find()=(%d, %v), want (0, false)", found, ok)
		}
	})

	t.Run("All", func(t *testing.T) {
		pulled := 0

		got := gchain.FromSeq(func(yield func(int) bool) {
			for _, value := range []int{1, 2, 3, 4} {
				pulled++
				if !yield(value) {
					return
				}
			}
		}).All(func(value int) bool {
			return value < 3
		})

		if got {
			t.Fatal("All()=true, want false")
		}
		if pulled != 3 {
			t.Fatalf("pulled=%d, want 3", pulled)
		}
	})

	t.Run("Take", func(t *testing.T) {
		pulled := 0

		got := gchain.FromSeq(func(yield func(int) bool) {
			for _, value := range []int{1, 2, 3, 4} {
				pulled++
				if !yield(value) {
					return
				}
			}
		}).Take(2).ToSlice()

		if !reflect.DeepEqual(got, []int{1, 2}) {
			t.Fatalf("Take(2).ToSlice()=%v, want [1 2]", got)
		}
		if pulled != 2 {
			t.Fatalf("pulled=%d, want 2", pulled)
		}
	})

	t.Run("TakeZero", func(t *testing.T) {
		pulled := 0

		got := gchain.FromSeq(func(yield func(int) bool) {
			for _, value := range []int{1, 2, 3} {
				pulled++
				if !yield(value) {
					return
				}
			}
		}).Take(0).ToSlice()

		if len(got) != 0 {
			t.Fatalf("Take(0).ToSlice()=%v, want empty", got)
		}
		if pulled != 0 {
			t.Fatalf("pulled=%d, want 0", pulled)
		}
	})
}

func TestTakeDropWhileOperations(t *testing.T) {
	t.Run("TakeWhile", func(t *testing.T) {
		pulled := 0
		calls := 0

		got := gchain.FromSeq(func(yield func(int) bool) {
			for _, value := range []int{1, 2, 3, 4} {
				pulled++
				if !yield(value) {
					return
				}
			}
		}).
			TakeWhile(func(value int) bool {
				calls++
				return value < 3
			}).
			ToSlice()

		if !reflect.DeepEqual(got, []int{1, 2}) {
			t.Fatalf("TakeWhile().ToSlice()=%v, want [1 2]", got)
		}
		if pulled != 3 || calls != 3 {
			t.Fatalf("pulled=%d calls=%d, want 3 and 3", pulled, calls)
		}
	})

	t.Run("DropWhile", func(t *testing.T) {
		calls := 0

		got := gchain.From([]int{1, 2, 3, 1}).
			DropWhile(func(value int) bool {
				calls++
				return value < 3
			}).
			ToSlice()

		if !reflect.DeepEqual(got, []int{3, 1}) {
			t.Fatalf("DropWhile().ToSlice()=%v, want [3 1]", got)
		}
		if calls != 3 {
			t.Fatalf("DropWhile calls=%d, want 3", calls)
		}
	})

	t.Run("AllPrefixMatches", func(t *testing.T) {
		got := gchain.From([]int{1, 2}).
			DropWhile(func(value int) bool {
				return value < 3
			}).
			ToSlice()

		if len(got) != 0 {
			t.Fatalf("DropWhile().ToSlice()=%v, want empty", got)
		}
	})
}

func TestChunkOperation(t *testing.T) {
	t.Run("ToSlice", func(t *testing.T) {
		pulled := 0

		chain := gchain.FromSeq(func(yield func(int) bool) {
			for _, value := range []int{1, 2, 3, 4, 5} {
				pulled++
				if !yield(value) {
					return
				}
			}
		}).Chunk(2)

		if pulled != 0 {
			t.Fatalf("Chunk ran before terminal operation: pulled=%d", pulled)
		}

		got := chain.ToSlice()
		want := [][]int{{1, 2}, {3, 4}, {5}}

		if !reflect.DeepEqual(got, want) {
			t.Fatalf("Chunk(2).ToSlice()=%v, want %v", got, want)
		}
		if pulled != 5 {
			t.Fatalf("pulled=%d, want 5", pulled)
		}
	})

	t.Run("IndependentSlices", func(t *testing.T) {
		got := gchain.From([]int{1, 2, 3, 4}).
			Chunk(2).
			ToSlice()

		got[0][0] = 99
		if !reflect.DeepEqual(got, [][]int{{99, 2}, {3, 4}}) {
			t.Fatalf("Chunk returned shared slices: got %v", got)
		}
	})

	t.Run("EarlyStop", func(t *testing.T) {
		pulled := 0

		got := gchain.FromSeq(func(yield func(int) bool) {
			for _, value := range []int{1, 2, 3, 4} {
				pulled++
				if !yield(value) {
					return
				}
			}
		}).
			Chunk(2).
			Take(1).
			ToSlice()

		if !reflect.DeepEqual(got, [][]int{{1, 2}}) {
			t.Fatalf("Chunk(2).Take(1).ToSlice()=%v, want [[1 2]]", got)
		}
		if pulled != 2 {
			t.Fatalf("pulled=%d, want 2", pulled)
		}
	})

	t.Run("ChunkMethods", func(t *testing.T) {
		chunks := gchain.From([]int{1, 2, 3, 4, 5}).
			Chunk(2).
			Filter(func(chunk []int) bool {
				return len(chunk) == 2
			})

		first, ok := chunks.First()
		if !reflect.DeepEqual(first, []int{1, 2}) || !ok {
			t.Fatalf("Chunks.First()=(%v, %v), want ([1 2], true)", first, ok)
		}
		if count := chunks.Count(); count != 2 {
			t.Fatalf("Chunks.Count()=%d, want 2", count)
		}

		sums := chunks.
			Drop(1).
			Map(func(chunk []int) int {
				return chunk[0] + chunk[1]
			}).
			ToSlice()
		if !reflect.DeepEqual(sums, []int{7}) {
			t.Fatalf("Chunks.Map().ToSlice()=%v, want [7]", sums)
		}

		seen := make([][]int, 0)
		chunks.Take(1).ForEach(func(chunk []int) {
			seen = append(seen, chunk)
		})
		if !reflect.DeepEqual(seen, [][]int{{1, 2}}) {
			t.Fatalf("Chunks.ForEach() saw %v, want [[1 2]]", seen)
		}
	})
}

func TestFlatMapDistinctByAndReverse(t *testing.T) {
	t.Run("FlatMap", func(t *testing.T) {
		calls := 0

		chain := gchain.From([]int{1, 2, 3}).FlatMap(func(value int) []int {
			calls++
			return []int{value, value * 10}
		})

		if calls != 0 {
			t.Fatalf("FlatMap ran before terminal operation: calls=%d", calls)
		}

		got := chain.ToSlice()
		want := []int{1, 10, 2, 20, 3, 30}

		if !reflect.DeepEqual(got, want) {
			t.Fatalf("FlatMap().ToSlice()=%v, want %v", got, want)
		}
		if calls != 3 {
			t.Fatalf("FlatMap calls=%d, want 3", calls)
		}
	})

	t.Run("DistinctBy", func(t *testing.T) {
		calls := 0

		got := gchain.From([]int{1, 2, 1, 3, 2, 4}).
			DistinctBy(func(value int) int {
				calls++
				return value
			}).
			ToSlice()

		want := []int{1, 2, 3, 4}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("DistinctBy().ToSlice()=%v, want %v", got, want)
		}
		if calls != 6 {
			t.Fatalf("DistinctBy calls=%d, want 6", calls)
		}
	})

	t.Run("Reverse", func(t *testing.T) {
		got := gchain.From([]int{1, 2, 3}).Reverse().ToSlice()
		want := []int{3, 2, 1}

		if !reflect.DeepEqual(got, want) {
			t.Fatalf("Reverse().ToSlice()=%v, want %v", got, want)
		}
	})

	t.Run("ReversePairs", func(t *testing.T) {
		got := gchain.FromSeq2(func(yield func(string, int) bool) {
			for _, pair := range []struct {
				key   string
				value int
			}{
				{key: "a", value: 1},
				{key: "b", value: 2},
				{key: "c", value: 3},
			} {
				if !yield(pair.key, pair.value) {
					return
				}
			}
		}).Reverse().ToPairs()

		want := []gchain.Pair[string, int]{
			{Key: "c", Value: 3},
			{Key: "b", Value: 2},
			{Key: "a", Value: 1},
		}

		if !reflect.DeepEqual(got, want) {
			t.Fatalf("Reverse pairs=%v, want %v", got, want)
		}
	})
}

func TestGroupByDeferredBufferingAndSinglePass(t *testing.T) {
	pulled := 0
	keyCalls := 0

	groups := gchain.FromSeq(func(yield func(int) bool) {
		for _, value := range []int{1, 2, 3, 4} {
			pulled++
			if !yield(value) {
				return
			}
		}
	}).
		Filter(func(value int) bool {
			return value > 1
		}).
		GroupBy(func(value int) string {
			keyCalls++
			if value%2 == 0 {
				return "even"
			}
			return "odd"
		})

	if pulled != 0 || keyCalls != 0 {
		t.Fatalf("GroupBy ran before terminal operation: pulled=%d keyCalls=%d", pulled, keyCalls)
	}

	got := groups.
		MapValues(func(values []int) int {
			return len(values)
		}).
		ToMap()
	want := map[string]int{"even": 2, "odd": 1}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("group counts=%v, want %v", got, want)
	}
	if pulled != 4 {
		t.Fatalf("pulled=%d, want 4", pulled)
	}
	if keyCalls != 3 {
		t.Fatalf("keyCalls=%d, want 3", keyCalls)
	}
}

func TestGroupsOperations(t *testing.T) {
	groups := gchain.From([]testUser{
		{ID: "1", Team: "platform", Active: true},
		{ID: "2", Team: "edge", Active: true},
		{ID: "3", Team: "platform", Active: false},
	}).GroupBy(func(user testUser) string {
		return user.Team
	})

	counts := groups.
		Filter(func(team string, users []testUser) bool {
			return len(users) > 1
		}).
		MapValues(func(users []testUser) int {
			return len(users)
		}).
		ToMap()
	if !reflect.DeepEqual(counts, map[string]int{"platform": 2}) {
		t.Fatalf("filtered group counts=%v, want platform=2", counts)
	}

	keys := groups.Keys().ToSlice()
	slices.Sort(keys)
	if !reflect.DeepEqual(keys, []string{"edge", "platform"}) {
		t.Fatalf("group keys=%v, want [edge platform]", keys)
	}

	labels := groups.Map(func(team string, users []testUser) string {
		return team + ":" + strings.Repeat("x", len(users))
	}).ToSlice()
	slices.Sort(labels)
	if !reflect.DeepEqual(labels, []string{"edge:x", "platform:xx"}) {
		t.Fatalf("group labels=%v, want edge/platform labels", labels)
	}

	pairs := groups.ToPairs()
	slices.SortFunc(pairs, func(left, right gchain.Pair[string, []testUser]) int {
		return strings.Compare(left.Key, right.Key)
	})
	if len(pairs) != 2 || pairs[0].Key != "edge" || len(pairs[1].Value) != 2 {
		t.Fatalf("group pairs=%v, want sorted edge/platform groups", pairs)
	}

	if groups.Count() != 2 {
		t.Fatalf("groups.Count()=%d, want 2", groups.Count())
	}

	seen := make(map[string]int)
	groups.ForEach(func(team string, users []testUser) {
		seen[team] = len(users)
	})
	if !reflect.DeepEqual(seen, map[string]int{"edge": 1, "platform": 2}) {
		t.Fatalf("groups.ForEach() saw %v, want edge/platform counts", seen)
	}
}

func TestCollectors(t *testing.T) {
	users := []testUser{
		{ID: "1", Name: "old", Team: "platform", Active: true},
		{ID: "2", Name: "bee", Team: "edge", Active: true},
		{ID: "1", Name: "new", Team: "platform", Active: false},
	}

	names := gchain.From(users).ToMap(func(user testUser) (string, string) {
		return user.ID, user.Name
	})
	if !reflect.DeepEqual(names, map[string]string{"1": "new", "2": "bee"}) {
		t.Fatalf("ToMap()=%v, want last value for duplicate key", names)
	}

	byID := gchain.From(users).ToMapBy(func(user testUser) string {
		return user.ID
	})
	if byID["1"].Name != "new" {
		t.Fatalf("ToMapBy duplicate key kept %q, want new", byID["1"].Name)
	}

	groups := gchain.From(users).ToGroups(func(user testUser) string {
		return user.Team
	})
	if len(groups["platform"]) != 2 || len(groups["edge"]) != 1 {
		t.Fatalf("ToGroups()=%v, want platform=2 edge=1", groups)
	}

	sum := gchain.From([]int{1, 2, 3}).Reduce(10, func(acc, value int) int {
		return acc + value
	})
	if sum != 16 {
		t.Fatalf("Reduce()=%d, want 16", sum)
	}

	counts := gchain.From(users).CountBy(func(user testUser) string {
		return user.Team
	})
	if !reflect.DeepEqual(counts, map[string]int{"platform": 2, "edge": 1}) {
		t.Fatalf("CountBy()=%v, want platform=2 edge=1", counts)
	}
}

func TestSourceMutationBeforeTerminal(t *testing.T) {
	values := []int{1, 2}
	chain := gchain.From(values)

	values[0] = 9

	got := chain.ToSlice()
	if !reflect.DeepEqual(got, []int{9, 2}) {
		t.Fatalf("ToSlice()=%v, want mutated element visible", got)
	}
}

func TestNilSliceAndMapProduceEmptyResults(t *testing.T) {
	var values []int
	gotSlice := gchain.From(values).ToSlice()

	if gotSlice == nil {
		t.Fatal("ToSlice() returned nil, want empty non-nil slice")
	}
	if len(gotSlice) != 0 {
		t.Fatalf("ToSlice() length=%d, want 0", len(gotSlice))
	}

	var valuesByKey map[string]int
	gotMap := gchain.FromMap(valuesByKey).ToMap()

	if gotMap == nil {
		t.Fatal("ToMap() returned nil, want empty non-nil map")
	}
	if len(gotMap) != 0 {
		t.Fatalf("ToMap() length=%d, want 0", len(gotMap))
	}
}

func TestNilFunctionsPanic(t *testing.T) {
	mustPanic(t, func() { _ = gchain.From([]int{}).FlatMap[int](nil) })
	mustPanic(t, func() { _ = gchain.From([]int{}).DistinctBy[int](nil) })
	mustPanic(t, func() { _ = gchain.From([]int{}).Map[int](nil) })
	mustPanic(t, func() { _ = gchain.From([]int{}).MapIndexed[int](nil) })
	mustPanic(t, func() { _ = gchain.From([]int{}).MapFilter[int](nil) })
	mustPanic(t, func() { _ = gchain.From([]int{}).MapFilterIndexed[int](nil) })
	mustPanic(t, func() { _ = gchain.From([]int{}).Filter(nil) })
	mustPanic(t, func() { _ = gchain.From([]int{}).FilterIndexed(nil) })
	mustPanic(t, func() { _ = gchain.From([]int{}).Take(-1) })
	mustPanic(t, func() { _ = gchain.From([]int{}).Drop(-1) })
	mustPanic(t, func() { _ = gchain.From([]int{}).TakeWhile(nil) })
	mustPanic(t, func() { _ = gchain.From([]int{}).DropWhile(nil) })
	mustPanic(t, func() { _ = gchain.From([]int{}).Chunk(0) })
	mustPanic(t, func() { _ = gchain.From([]int{}).Chunk(-1) })
	mustPanic(t, func() { _ = gchain.From([]int{}).GroupBy[int](nil) })
	mustPanic(t, func() { _ = gchain.From([]int{}).ToMap[int, int](nil) })
	mustPanic(t, func() { _ = gchain.From([]int{}).ToMapBy[int](nil) })
	mustPanic(t, func() { _ = gchain.From([]int{}).ToGroups[int](nil) })
	mustPanic(t, func() { _ = gchain.From([]int{}).CountBy[int](nil) })
	mustPanic(t, func() { _, _ = gchain.From([]int{}).Find(nil) })
	mustPanic(t, func() { _ = gchain.From([]int{}).Any(nil) })
	mustPanic(t, func() { _ = gchain.From([]int{}).All(nil) })
	mustPanic(t, func() { _ = gchain.From([]int{}).Reduce(0, nil) })
	mustPanic(t, func() { gchain.From([]int{}).ForEach(nil) })

	chunks := gchain.From([]int{1}).Chunk(1)
	mustPanic(t, func() { _ = chunks.Filter(nil) })
	mustPanic(t, func() { _ = chunks.Map[int](nil) })
	mustPanic(t, func() { _ = chunks.Take(-1) })
	mustPanic(t, func() { _ = chunks.Drop(-1) })
	mustPanic(t, func() { chunks.ForEach(nil) })

	pairs := gchain.FromMap(map[string]int{})
	mustPanic(t, func() { _ = pairs.Filter(nil) })
	mustPanic(t, func() { _ = pairs.FilterKeys(nil) })
	mustPanic(t, func() { _ = pairs.FilterValues(nil) })
	mustPanic(t, func() { _ = pairs.Map[int](nil) })
	mustPanic(t, func() { _ = pairs.MapKeys[string](nil) })
	mustPanic(t, func() { _ = pairs.MapValues[int](nil) })
	mustPanic(t, func() { pairs.ForEach(nil) })

	groups := gchain.From([]int{1}).GroupBy(func(value int) int { return value })
	mustPanic(t, func() { _ = groups.Filter(nil) })
	mustPanic(t, func() { _ = groups.Map[int](nil) })
	mustPanic(t, func() { _ = groups.MapValues[int](nil) })
	mustPanic(t, func() { groups.ForEach(nil) })
}

func TestPairsOperations(t *testing.T) {
	got := gchain.FromMap(map[string]int{
		"a": 1,
		"b": 2,
		"c": 3,
	}).
		Filter(func(key string, value int) bool {
			return value%2 == 1
		}).
		MapKeys(strings.ToUpper).
		MapValues(func(value int) string {
			return strings.Repeat("x", value)
		}).
		ToMap()
	want := map[string]string{"A": "x", "C": "xxx"}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("pairs ToMap()=%v, want %v", got, want)
	}

	byKey := gchain.FromMap(map[string]int{"apple": 1, "banana": 2, "apricot": 3}).
		FilterKeys(func(key string) bool {
			return strings.HasPrefix(key, "ap")
		}).
		ToMap()
	if !reflect.DeepEqual(byKey, map[string]int{"apple": 1, "apricot": 3}) {
		t.Fatalf("FilterKeys().ToMap()=%v, want apple/apricot", byKey)
	}

	byValue := gchain.FromMap(map[string]int{"a": 1, "b": 2, "c": 3}).
		FilterValues(func(value int) bool {
			return value > 1
		}).
		ToMap()
	if !reflect.DeepEqual(byValue, map[string]int{"b": 2, "c": 3}) {
		t.Fatalf("FilterValues().ToMap()=%v, want b/c", byValue)
	}

	keys := gchain.FromMap(map[string]int{"b": 2, "a": 1}).Keys().ToSlice()
	slices.Sort(keys)
	if !reflect.DeepEqual(keys, []string{"a", "b"}) {
		t.Fatalf("Keys()=%v, want [a b]", keys)
	}

	values := gchain.FromMap(map[string]int{"a": 1, "b": 2}).Values().ToSlice()
	slices.Sort(values)
	if !reflect.DeepEqual(values, []int{1, 2}) {
		t.Fatalf("Values()=%v, want [1 2]", values)
	}

	mapped := gchain.FromMap(map[string]int{"a": 1}).
		Map(func(key string, value int) string {
			return key + strings.Repeat("!", value)
		}).
		ToSlice()
	if !reflect.DeepEqual(mapped, []string{"a!"}) {
		t.Fatalf("Pairs.Map().ToSlice()=%v, want [a!]", mapped)
	}

	pairs := gchain.FromMap(map[string]int{"a": 1, "b": 2}).ToPairs()
	slices.SortFunc(pairs, func(left, right gchain.Pair[string, int]) int {
		return strings.Compare(left.Key, right.Key)
	})
	if !reflect.DeepEqual(pairs, []gchain.Pair[string, int]{
		{Key: "a", Value: 1},
		{Key: "b", Value: 2},
	}) {
		t.Fatalf("ToPairs()=%v, want sorted pairs", pairs)
	}
}

func TestFromSeqSingleUseSource(t *testing.T) {
	values := []int{1, 2, 3}
	index := 0

	chain := gchain.FromSeq(func(yield func(int) bool) {
		for index < len(values) {
			value := values[index]
			index++
			if !yield(value) {
				return
			}
		}
	})

	first := chain.ToSlice()
	second := chain.ToSlice()

	if !reflect.DeepEqual(first, []int{1, 2, 3}) {
		t.Fatalf("first ToSlice()=%v, want [1 2 3]", first)
	}
	if len(second) != 0 {
		t.Fatalf("second ToSlice()=%v, want empty", second)
	}
}

func TestSeqRangeAdapters(t *testing.T) {
	values := make([]int, 0)
	for value := range gchain.From([]int{1, 2, 3}).Filter(func(value int) bool {
		return value > 1
	}).Seq() {
		values = append(values, value)
	}
	if !reflect.DeepEqual(values, []int{2, 3}) {
		t.Fatalf("range Seq values=%v, want [2 3]", values)
	}

	got := make(map[string]int)
	for key, value := range gchain.FromMap(map[string]int{"a": 1, "b": 2}).Seq2() {
		got[key] = value
	}
	if !reflect.DeepEqual(got, map[string]int{"a": 1, "b": 2}) {
		t.Fatalf("range Seq2 values=%v, want original map", got)
	}
}

func mustPanic(t *testing.T, f func()) {
	t.Helper()

	defer func() {
		if recover() == nil {
			t.Fatal("expected panic")
		}
	}()

	f()
}
