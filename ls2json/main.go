package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"
  "strings"

	vfs "github.com/kazu/vfs-index"
	"github.com/vbauerster/mpb/v5"
)

const Usage string = `
Usage ls2json dir <flags>
	flags:
		--output	output json filename
		--newer		output only newser files mtime.
`

func main() {

	if len(os.Args) < 2 {
		fmt.Println(Usage)
		return
	}
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
	dirs := os.Args[1]

  records := make([]FileRecord, 0)

  for _, dir := range strings.Split(dirs, ",") {
	  if recs, e := lsR(dir, newer); e == nil {
      records = append(records, recs...)
    }else{
      log.Println(e)
    }
  }

	result, e := json.Marshal(records)
	if e != nil {
		log.Println(e)
		return
	}
	if output == "" {
		fmt.Println(string(result))
		return
	}

	f, e := os.Create(output)
	if e != nil {
		fmt.Println(e)
		return
	}
	defer f.Close()
	f.Write(result)

}

type FileRecord struct {
	Drive   string
	Inode   uint64
	Dir     string
	Name    string
	ModTime int64
	Size    int64
}

func lsR(dir string, cond time.Time) (result []FileRecord, e error) {
	//bar := vfs.PBar.Add("finding file"
	mbar := vfs.NewProgressBar(mpb.WithOutput(os.Stderr))

	total := 65536
	bar := mbar.Add("search file", total)

	defer func() {
		bar.SetTotal(bar.Current(), true)
		mbar.Done()
	}()

	e = filepath.Walk(dir,
		func(path string, info os.FileInfo, err error) error {
			if err != nil {
				log.Println(err)
				if info != nil && info.IsDir() {
					return filepath.SkipDir
				}
				return nil

			}
			if info.IsDir() || info.ModTime().Before(cond) {
				return nil
			}

      if strings.HasPrefix(filepath.Base(path), "._") {
        return nil
      }

			rel, _ := filepath.Rel(dir, path)
			bar.IncrBy(1)
			if bar.Current() >= (int64(total) - 10) {
				total += (total / 2)
				bar.SetTotal(int64(total), false)

			}
			result = append(result, FileRecord{
				Drive:   dir,
				Inode:   vfs.GetInode(info),
				Dir:     filepath.Dir(rel),
				Name:    filepath.Base(rel),
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
