package ext_test

import (
	"math/rand"
	"slices"
	"testing"
	"time"
)

type Integer interface {
	~int | ~int8 | ~int16 | ~int32 | ~int64
}

func MakeRndSlice[T Integer](seed int64, cnt int) (ret []T) {

	rnd := rand.New(rand.NewSource(seed))

	ret = make([]T, cnt)
	var v int32

	for i := 0; i < cnt; i++ {
		v = rnd.Int31()
		ret[i] = T(v)
	}

	return ret
}

func BenchmarkMapSlice(b *testing.B) {

	cnt := 10000
	seed1 := int64(12345)
	seed2 := int64(45672)

	int32s := MakeRndSlice[int32](seed1, cnt)
	finds := MakeRndSlice[int32](seed2, cnt)

	b.Run("map[int32]bool 10000", func(b *testing.B) {
		b.StopTimer()

		m := map[int32]bool{}
		b.StartTimer()
		for _, v := range int32s {
			m[v] = true
		}
		b.StopTimer()

		var got bool
		_ = got
		for i := 0; i < b.N; i++ {
			b.StartTimer()
			for _, v := range finds {
				got = m[v]
			}
			b.StopTimer()
		}
	})

	b.Run("map[int]struct{} 10000", func(b *testing.B) {
		b.StopTimer()

		m := map[int32]struct{}{}

		b.StartTimer()
		for _, v := range int32s {
			m[v] = struct{}{}
		}
		b.StopTimer()

		var got bool
		_ = got
		for i := 0; i < b.N; i++ {
			b.StartTimer()
			for _, v := range finds {
				_, got = m[v]
			}
			b.StopTimer()
		}
	})

	// b.Run("map[int]bool 10000", func(b *testing.B) {
	// })

}

// 総数からフィルタしたサンプルに対してランダムな target を探すベンチ
func BenchmarkSearchTarget(b *testing.B) {
	b.Cleanup(func() { time.Sleep(time.Second) })

	siz := 10000   // 総数
	sample := 5000 // サンプル
	count := 100   // 検索回数

	setup := func(siz, sample int) (all, samples []int) {
		all = make([]int, siz)
		for i := range siz {
			all[i] = i
		}
		// 並びを適当にする
		for i := len(all) - 1; i > 0; i-- {
			j := rand.Intn(i + 1)
			all[i], all[j] = all[j], all[i]
		}
		// サンプル
		samples = all[:sample]
		return
	}

	// ソート済みスライスを作る
	makeSort := func(s []int) []int {
		r := slices.Clone(s)
		slices.Sort(s)
		return r
	}

	// マップを作る
	makeMapper := func(useMap bool, s []int) (r map[int]struct{}) {

		if useMap {
			r = make(map[int]struct{}, len(s))
		} else {
			r = map[int]struct{}{}
		}
		for _, m := range s {
			r[m] = struct{}{}
		}
		return r
	}
	makeMapBool := func(useMap bool, s []int) (r map[int]bool) {
		if useMap {
			r = make(map[int]bool, len(s))
		} else {
			r = map[int]bool{}
		}
		for _, m := range s {
			r[m] = true
		}
		return r
	}

	b.Run("slice", func(b *testing.B) {
		all, samples := setup(siz, sample)
		b.ResetTimer()
		for range b.N {
			src := makeSort(samples)
			for range count {
				target := all[rand.Intn(len(all))]
				if _, ok := slices.BinarySearch(src, target); ok {
					// ...
				}
			}
		}
	})
	b.Run("map", func(b *testing.B) {
		all, samples := setup(siz, sample)
		b.ResetTimer()
		for range b.N {
			src := makeMapper(true, samples)
			for range count {
				target := all[rand.Intn(len(all))]
				if _, ok := src[target]; ok {
					// ...
				}
			}
		}
	})
	// b.Run("map2", func(b *testing.B) {
	// 	all, samples := setup(siz, sample)
	// 	b.ResetTimer()
	// 	for range b.N {
	// 		src := makeMapper(false, samples)
	// 		for range count {
	// 			target := all[rand.Intn(len(all))]
	// 			if _, ok := src[target]; ok {
	// 				// ...
	// 			}
	// 		}
	// 	}
	// })
	b.Run("map bool", func(b *testing.B) {
		all, samples := setup(siz, sample)
		b.ResetTimer()
		for range b.N {
			src := makeMapBool(true, samples)
			for range count {
				target := all[rand.Intn(len(all))]
				if _, ok := src[target]; ok {
					// ...
				}
			}
		}
	})
	// b.Run("map bool2", func(b *testing.B) {
	// 	all, samples := setup(siz, sample)
	// 	b.ResetTimer()
	// 	for range b.N {
	// 		src := makeMapBool(false, samples)
	// 		for range count {
	// 			target := all[rand.Intn(len(all))]
	// 			if _, ok := src[target]; ok {
	// 				// ...
	// 			}
	// 		}
	// 	}
	// })

}
