package ext

import (
	"os"
	"path/filepath"

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
