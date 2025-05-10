package awalk

import (
	"io/fs"
	"os"
	"path/filepath"

	"github.com/kazu/cmd/ext"
)

// var SkipDir error = fs.SkipDir
// var SkipAll error = fs.SkipAll

// type WalkerFn func(path string, info os.FileInfo, err error, opts ...ext.OptFunc[walkerOpt]) error

// type walkerOpt struct {
// 	max      uint32       `default:"32"`
// 	runnings atomic.Int32 `default:"0"`
// 	ctx      context.Context
// 	can      context.CancelFunc
// }

// func incRunning(inc int32) ext.OptFunc[walkerOpt] {

// 	return func(opt *walkerOpt) ext.OptFunc[walkerOpt] {
// 		opt.runnings.Add(inc)

// 		return nil
// 	}
// }

// func Walk(root string, fn WalkerFn, opts ...ext.OptFunc[walkerOpt]) error {
// 	info, err := os.Lstat(root)
// 	if err != nil {
// 		err = fn(root, nil, err, opts...)
// 	} else {
// 		err = walk(root, info, fn, opts...)
// 	}
// 	if err == SkipDir || err == SkipAll {
// 		return nil
// 	}
// 	return err
// }

// func walk(path string, info os.FileInfo, walkFn WalkerFn, opts ...ext.OptFunc[walkerOpt]) error {

// 	if !info.IsDir() {
// 		return walkFn(path, info, nil, opts...)
// 	}

// 	names, err := readDirNames(path)
// 	err1 := walkFn(path, info, err, opts...)
// 	if err != nil || err1 != nil {
// 		return err1
// 	}

// 	wopt := ext.NewWithDefault[walkerOpt]()
// 	ext.HandleOpt(wopt, opts...)

// 	nopts := make([]ext.OptFunc[walkerOpt], len(opts), len(opts)+len(names))
// 	copy(nopts, opts)

// 	canRun := wopt.max - uint32(wopt.runnings.Load())
// 	_ = canRun

// 	// if canRun <= 0 {
// 	// 	return _wark_fn(path, names, info, walkFn, nopts...)
// 	// }

// 	// var wg sync.WaitGroup

// 	// wg.Add(1)
// 	// var result sync.AtomicValue[error]
// 	ctx, can := context.WithCancel(context.Background())
// 	var atomicErr atomic.Value
// 	atomicErr.Store(nil)

// 	for name := range slices.Values(names) {
// 		filename := filepath.Join(path, name)
// 		fileInfo, err := os.Lstat(filename)
// 		if err != nil {
// 			if err := walkFn(filename, fileInfo, err, nopts...); err != nil && err != SkipDir {
// 				return err
// 			}
// 		}
// 		if canRun > 0 {
// 			wopt.runnings.Add(1)
// 			canRun--
// 			go func(ctx context.Context, can func(), atomicErr *atomic.Value) {
// 				defer func() {
// 					nopts = append(nopts, incRunning(-1))
// 				}()
// 				err := walk(filename, fileInfo, walkFn, nopts...)
// 				if err != nil {
// 					if !fileInfo.IsDir() || err != SkipDir {
// 						can()
// 					}
// 				}
// 			}(ctx, can, &atomicErr)
// 		} else {
// 			err = walk(filename, fileInfo, walkFn, nopts...)
// 			if err != nil {
// 				if !fileInfo.IsDir() || err != SkipDir {
// 					return err
// 				}
// 			}

// 		}

// 	}

// 	return nil
// }

// func _wark_fn(path string, names []string, info os.FileInfo, walkFn WalkerFn, opts ...ext.OptFunc[walkerOpt]) error {
// 	for name := range slices.Values(names) {
// 		filename := filepath.Join(path, name)
// 		fileInfo, err := os.Lstat(filename)

// 		if err != nil {
// 			if err := walkFn(filename, fileInfo, err, opts...); err != nil && err != SkipDir {
// 				return err
// 			}
// 		}
// 		err = walk(filename, fileInfo, walkFn, opts...)
// 		if err != nil {
// 			if !fileInfo.IsDir() || err != SkipDir {
// 				return err
// 			}
// 		}
// 	}
// 	return nil

// }

// // the readDirNames function below was taken from the original
// // implementation (see https://golang.org/src/path/filepath/path.go)
// // but has sorting removed (sorting doesn't make sense
// // in concurrent execution, anyway)

// // readDirNames reads the directory named by dirname and returns
// // a sorted list of directory entry names.
//
//	func readDirNames(dirname string) ([]string, error) {
//		f, err := os.Open(dirname)
//		if err != nil {
//			return nil, err
//		}
//		names, err := f.Readdirnames(-1)
//		f.Close()
//		if err != nil {
//			return nil, err
//		}
//		slices.Sort(names)
//		return names, nil
//	}
type WalkerFn func(path string, info fs.FileInfo, err error) error

type waklReq struct {
	path string
	info fs.FileInfo
}

type walkOpt struct {
	max uint32 `default:"32"`
}

func Max(max int) ext.OptFunc[walkOpt] {
	return func(opt *walkOpt) ext.OptFunc[walkOpt] {
		opt.max = uint32(max)
		return nil
	}
}

func Walk(root string, max int, fn WalkerFn) {
	ch := walk(root, Max(max))

	for req := range ch {
		fn(req.path, req.info, nil)
	}
}

func walk(root string, opt ...ext.OptFunc[walkOpt]) chan waklReq {
	wOpt := ext.NewWithDefault[walkOpt]()
	ext.HandleOpt(wOpt, opt...)
	ch := make(chan waklReq, wOpt.max)
	go func() {
		filepath.Walk(root, func(path string, finfo os.FileInfo, _ error) (err error) {
			ch <- waklReq{path: path, info: finfo}
			return
		})
		defer close(ch)
	}()

	return ch

}
