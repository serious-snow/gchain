package gchain_test

import (
	"testing"

	"github.com/serious-snow/gchain"
)

var (
	benchmarkSliceSink  []int
	benchmarkMapSink    map[int]int
	benchmarkGroupsSink map[int][]int
)

func BenchmarkChainToSliceFromSlice(b *testing.B) {
	values := benchmarkInts(1024)

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		benchmarkSliceSink = gchain.From(values).ToSlice()
	}
}

func BenchmarkChainToSliceAfterFilter(b *testing.B) {
	values := benchmarkInts(1024)

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		benchmarkSliceSink = gchain.From(values).
			Filter(func(value int) bool {
				return value%2 == 0
			}).
			ToSlice()
	}
}

func BenchmarkChainToMapFromSlice(b *testing.B) {
	values := benchmarkInts(1024)

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		benchmarkMapSink = gchain.From(values).ToMap(func(value int) (int, int) {
			return value, value * 2
		})
	}
}

func BenchmarkPairsToMapFromMap(b *testing.B) {
	values := benchmarkIntMap(1024)

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		benchmarkMapSink = gchain.FromMap(values).ToMap()
	}
}

func BenchmarkGroupByToMap(b *testing.B) {
	values := benchmarkInts(1024)

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		benchmarkGroupsSink = gchain.From(values).
			GroupBy(func(value int) int {
				return value % 16
			}).
			ToMap()
	}
}

func BenchmarkReverseToSlice(b *testing.B) {
	values := benchmarkInts(1024)

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		benchmarkSliceSink = gchain.From(values).
			Reverse().
			ToSlice()
	}
}

func benchmarkInts(count int) []int {
	values := make([]int, count)
	for index := range values {
		values[index] = index
	}
	return values
}

func benchmarkIntMap(count int) map[int]int {
	values := make(map[int]int, count)
	for index := 0; index < count; index++ {
		values[index] = index
	}
	return values
}
