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
