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

	sum := gchain.From([]int{1, 2, 3}).Reduce(10, func(acc int, value int) int {
		return acc + value
	})
	if sum != 16 {
		t.Fatalf("Reduce()=%d, want 16", sum)
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
	mustPanic(t, func() { _ = gchain.From([]int{}).Filter(nil) })
	mustPanic(t, func() { _ = gchain.From([]int{}).Take(-1) })
	mustPanic(t, func() { _ = gchain.From([]int{}).Drop(-1) })
	mustPanic(t, func() { _ = gchain.From([]int{}).GroupBy[int](nil) })
	mustPanic(t, func() { _ = gchain.From([]int{}).ToMap[int, int](nil) })
	mustPanic(t, func() { _ = gchain.From([]int{}).ToMapBy[int](nil) })
	mustPanic(t, func() { _ = gchain.From([]int{}).ToGroups[int](nil) })
	mustPanic(t, func() { _ = gchain.From([]int{}).Any(nil) })
	mustPanic(t, func() { _ = gchain.From([]int{}).All(nil) })
	mustPanic(t, func() { _ = gchain.From([]int{}).Reduce(0, nil) })
	mustPanic(t, func() { gchain.From([]int{}).ForEach(nil) })

	pairs := gchain.FromMap(map[string]int{})
	mustPanic(t, func() { _ = pairs.Filter(nil) })
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
