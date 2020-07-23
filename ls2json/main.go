package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	vfs "github.com/kazu/vfs-index"
)

const Usage string = `
Usage ls2json
`

func main() {

	// if len(os.Args) == 1 {
	// 	fmt.Println(Usage)
	// }
	var output string
	var check string
	flagcmd := flag.NewFlagSet("ls2json", flag.ExitOnError)
	flagcmd.StringVar(&output, "output", "", "output json filename")
	flagcmd.StringVar(&check, "newer", "", "only list new file after <newer>.")
	flagcmd.Parse(os.Args[2:])

	var newer time.Time
	if len(check) > 0 {
		info, e := os.Stat(check)
		if e != nil {
			log.Println(e)
			return
		}
		newer = info.ModTime()
	} else {
		newer = time.Time{}
	}

	var e error
	dir := os.Args[1]

	records, e := lsR(dir, newer)
	if e != nil {
		log.Println(e)
		return
	}
	//spew.Dump(records)

	result, e := json.Marshal(records)
	if e != nil {
		log.Println(e)
		return
	}
	fmt.Println(string(result))

}

type FileRecord struct {
	Drive   string
	Inode   uint64
	Path    string
	ModTime int64
	Size    int64
}

func lsR(dir string, cond time.Time) (result []FileRecord, e error) {

	e = filepath.Walk(dir,
		func(path string, info os.FileInfo, err error) error {
			if err != nil {
				log.Println(err)
				if info.IsDir() {
					return filepath.SkipDir
				}
				return nil

			}
			if info.IsDir() || info.ModTime().Before(cond) {
				return nil
			}
			//fmt.Println(path, info.Size())
			rel, _ := filepath.Rel(dir, path)
			result = append(result, FileRecord{
				Drive:   dir,
				Inode:   vfs.GetInode(info),
				Path:    rel,
				ModTime: info.ModTime().UnixNano(),
				Size:    info.Size(),
			})
			return nil
		})
	if e != nil {
		return
	}
	return
}
