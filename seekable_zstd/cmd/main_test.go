package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"testing"

	seekable "github.com/SaveTheRbtz/zstd-seekable-format-go/pkg"
	"github.com/kazu/cmd/ext"
	"github.com/klauspost/compress/zstd"
)

func Test_make_seekable(t *testing.T) {

	tmpDir := "../../tmp"

	f := ext.WithErr(os.Create(filepath.Join(tmpDir, "test.seekable.zst"))).Value
	enc := ext.WithErr(zstd.NewWriter(nil, zstd.WithEncoderLevel(zstd.SpeedFastest))).Value

	w := ext.WithErr(seekable.NewWriter(f, enc)).Value

	for i := 0; i < 1000; i++ {
		p := i * 10 % 100
		s := fmt.Sprintf("%d,%d,%d,%d,%d,%d\n", p+1, p+2, i*6+3, i*6+4, i*6+5, i*6+6)
		w.Write([]byte(s))
	}
	w.Close()

	r := ext.WithErr(seekable.NewReader(ext.WithErr(os.Open(filepath.Join(tmpDir, "test.seekable.zst"))).Value, ext.WithErr(zstd.NewReader(nil)).Value)).Value
	r.Seek(18, io.SeekStart)
	src := io.LimitReader(r, 18)

	f = ext.WithErr(os.Create(filepath.Join(tmpDir, "test.seekable2.zst"))).Value
	enc = ext.WithErr(zstd.NewWriter(nil, zstd.WithEncoderLevel(zstd.SpeedFastest))).Value

	w = ext.WithErr(seekable.NewWriter(f, enc)).Value

	io.Copy(w, src)
	w.Close()
	r.Close()

}
