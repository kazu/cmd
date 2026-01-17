package ext

import (
	"iter"
	"os"
	"path/filepath"
	"sync"

	"github.com/mcuadros/go-defaults"
)

func NewWithDefault[T any]() *T {

	t := new(T)
	defaults.SetDefaults(t)
	return t
}

type OptFunc[T any] func(*T) OptFunc[T]

func HandleOpt[T any](t *T, opts ...OptFunc[T]) (previous OptFunc[T]) {

	for _, opt := range opts {
		previous = opt(t)
	}

	return
}

func IsFile(info os.FileInfo) bool {
	return !info.IsDir()
}

func FilesR(dir string, cond func(info os.FileInfo) bool) (result []string) {
	filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if cond(info) {
			result = append(result, path)
		}
		return nil
	})
	return

}

func FileRSeq(dir string, cond func(info os.FileInfo) bool) iter.Seq[string] {

	return func(yield func(string) bool) {
		filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if cond(info) {
				if !yield(path) {
					return filepath.SkipDir
				}
			}
			return nil
		})
		return
	}
}

// MENTION: only < golang1.21
func OnceFunc(fn func()) func() {
	var once sync.Once

	return func() {
		once.Do(fn)
	}
}

func OnceValue[T any](fn func() T) func() T {

	var once sync.Once
	var result T

	return func() T {
		once.Do(func() {
			result = fn()
		})
		return result
	}
}

func OnceArgsValue[T any, A any](fn func(...A) T) func(...A) T {

	var once sync.Once
	var result T

	return func(arg ...A) T {
		once.Do(func() {
			result = fn(arg...)
		})
		return result
	}
}

func OnceArgValue[T any, A any](fn func(A) T) func(A) T {

	var once sync.Once
	var result T

	return func(arg A) T {
		once.Do(func() {
			result = fn(arg)
		})
		return result
	}
}

type paraCfg struct {
	max int
}

var _defaultParaCnf = &paraCfg{
	max: -1,
}

func Concurrent(max int) OptFunc[paraCfg] {
	return func(pc *paraCfg) OptFunc[paraCfg] {
		pc.max = max
		return nil
	}
}

func paraFn[T any](idx int, t T, fn func(i int, v T), ch chan struct{}, cfg *paraCfg) {

	fn(idx, t)
	<-ch

}

func Parallel[T any](it iter.Seq[T], fn func(i int, t T), opts ...OptFunc[paraCfg]) {

	var paraOpt *paraCfg

	paraOpt = _defaultParaCnf
	var ch chan struct{}

	if len(opts) > 0 {
		paraOpt = &paraCfg{}
		HandleOpt(paraOpt, opts...)
	}

	if paraOpt.max == -1 {
		ch = make(chan struct{}, 100)
	} else {
		ch = make(chan struct{}, paraOpt.max)
	}

	idx := 0
	for v := range it {
		ch <- struct{}{}
		go paraFn(idx, v, fn, ch, paraOpt)
		idx++
	}
	for {
		if len(ch) == 0 {
			break
		}
	}

}

type Result[T any] struct {
	Value T
	Err   error
}

func WithErr[T any](t T, e error) Result[T] {
	return Result[T]{
		Value: t,
		Err:   e,
	}
}
