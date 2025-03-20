package main

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"syscall"

	"github.com/kazu/cmd/ext"
	"github.com/kazu/cmd/h2clone"
	log "github.com/kazu/cmd/logger"
)

type CmdParam struct {
	DryRun      bool `default:"false"`
	Dir         string
	LogLevel    string   `default:"info"`
	LogName     string   `default:"stdout"`
	ExcludeDirs []string `default:"[]"`
	IsHelp      bool     `default:"false"`
}

const Usege = `Usage: h2clone [dir] [dryrun] [exclude=dir] [log_level=level] [log_name=name]`

func NewCmdParam(dir string, dryrun string, args ...string) *CmdParam {

	param := ext.NewWithDefault[CmdParam]()
	param.Dir = dir
	param.DryRun = dryrun == "true"

	for arg := range slices.Values(args) {
		if arg == "help" {
			param.IsHelp = true
			break
		}
		var key, value string

		if strings.Contains(arg, "=") {
			key = arg[:strings.Index(arg, "=")]
			value = arg[strings.Index(arg, "=")+1:]
		}
		switch key {
		case "exclude":
			param.ExcludeDirs = append(param.ExcludeDirs, value)
		case "dryrun":
			param.DryRun = value == "true"
		case "log_level":
			param.LogLevel = value
		case "log_name":
			param.LogName = value
		}
	}
	return param
}

func main() {

	dir, _ := os.Getwd()
	args := make([]string, len(os.Args)-1)

	if len(args) > 0 {
		copy(args, os.Args[1:])
	}

	dir = args[0]
	dryrun := args[1]

	cParam := NewCmdParam(dir, dryrun, args[2:]...)

	if cParam.IsHelp {
		fmt.Println(Usege)
		return
	}

	log.SetupLogger(cParam.LogName, cParam.LogLevel)
	hardlink_to_clone(cParam.Dir, cParam.DryRun, cParam.ExcludeDirs)
}

var _inode2paths = make(map[uint64][]string)
var _muinode2paths sync.Mutex

func ino2paths(ino uint64) []string {
	_muinode2paths.Lock()
	defer _muinode2paths.Unlock()
	return _inode2paths[ino]
}

func addIno2paths(ino uint64, path string) {
	_muinode2paths.Lock()
	defer _muinode2paths.Unlock()
	_inode2paths[ino] = append(_inode2paths[ino], path)
}

func isHardlkinkAndFile(bdir string, excludes ...string) func(info os.FileInfo) (ok bool) {
	return func(info os.FileInfo) (ok bool) {
		if !ext.IsFile(info) {
			return false
		}
		// fname := filepath.Join(bdir, info.Name())
		// dir := filepath.Dir(fname)

		// fmt.Printf("f=%s dir=%s ex=%v ok=%v\n",
		// 	info.Name(),
		// 	dir,
		// 	excludes,
		// 	slices.Contains(excludes, dir))
		// if slices.Contains(excludes, dir) {
		// 	return false
		// }
		ok = isHardlink(info)
		if ok {
			log.Debug("isHardlkinkAndFile: hardlinks")(func(attr log.LazyFunc) {
				attr(
					slog.String("file", info.Name()),
				)

			})
		}

		return
	}
}
func isHardlink(info os.FileInfo) bool {

	sys := info.Sys()
	if sys == nil {
		return false
	}
	stat, ok := sys.(*syscall.Stat_t)
	if !ok {
		return false
	}
	nlink := uint64(0)
	nlink = uint64(stat.Nlink)

	if nlink > 1 {
		log.Debug("isHardlink: link count")(func(attr log.LazyFunc) {
			attr(
				slog.String("file", info.Name()),
				slog.Uint64("cnt", nlink),
			)
		})

	}

	return nlink > 1

}

func getInode(path string) uint64 {
	info, err := os.Stat(path)
	if err != nil {
		return 0
	}
	return info.Sys().(*syscall.Stat_t).Ino
}

func file2linstat(opath string) (path string, ppath string) {
	path = opath
	ino := getInode(opath)
	if !slices.Contains(ino2paths(ino), opath) {
		addIno2paths(ino, opath)
	}
	ppath = ino2paths(ino)[0]
	if ppath == path {
		ppath = ""
	}
	return
}
func hardlink_to_clone(src string, isDryRun bool, excludes []string) {

	for file := range ext.FileRSeq(src, isHardlkinkAndFile(src, excludes...)) {
		if slices.Contains(excludes, filepath.Dir(file)) {
			continue
		}
		if slices.ContainsFunc(excludes, func(exclude string) bool {
			return strings.Contains(filepath.Dir(file), exclude)
		}) {
			continue
		}

		_, ppath := file2linstat(file)
		if ppath == "" {
			continue
		}
		var err error
		err = nil
		attrs := []slog.Attr{
			slog.String("src", ppath),
			slog.String("dst", file),
		}

		if !isDryRun {
			err = h2clone.CloneFile(ppath, file)
		} else {
			attrs = append(attrs, slog.String("dryrun", "true"))
		}
		if err != nil {
			attrs = append(attrs, slog.String("err", err.Error()))
		}

		log.Info("hardlink_to_clone")(func(attr log.LazyFunc) {
			attr(attrs...)
		})
	}

}
