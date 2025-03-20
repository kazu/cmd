package fastcomp_test

import (
	"io"
	"math/rand"
	"os"
	"strconv"
	"sync"
	"testing"

	"github.com/pierrec/lz4/v4"
	"github.com/stretchr/testify/assert"
)

func makeSampleData(size int) []byte {

	data := make([]byte, 0, size)

	for i := 0; i < size/4; i++ {
		v := rand.New(rand.NewSource(int64(i))).Int63()
		data = strconv.AppendInt(data, v, 10)
	}

	return data

}

type sampleDataInfo struct {
	bufs [][]byte
	used int
	mu   sync.Mutex
	wgr  sync.WaitGroup
	wgw  sync.WaitGroup
}

func DefaultSampleDataInfo() sampleDataInfo {

	return sampleDataInfo{
		bufs: make([][]byte, 0, 1024),
		used: 0,
	}

}

func (sam *sampleDataInfo) Init(chunkSize int) {

	if len(sam.bufs) > 0 {
		return
	}

	sam.bufs = append(sam.bufs, make([]byte, 0, chunkSize))
	return
}

func (sam *sampleDataInfo) waitAndRound() {

	a := true
	_ = a
	sam.mu.Lock()
	sam.used++
	sam.mu.Unlock()
	if len(sam.bufs) == cap(sam.bufs) && sam.used > 0 {
		sam.wgr.Done()
	}

	if len(sam.bufs) <= sam.used && len(sam.bufs) < cap(sam.bufs) {
		sam.wgw.Add(1)
		sam.wgw.Wait()
		a = false
	}

	if len(sam.bufs) <= sam.used {
		sam.used = 0
	}

}

func (sam *sampleDataInfo) Read(data []byte) (int, error) {

	l := len(sam.bufs[sam.used])
	copy(data, sam.bufs[sam.used])

	sam.waitAndRound()

	return l, nil
}

func sampleData(size int, chunkSize int) *sampleDataInfo {

	data := DefaultSampleDataInfo()
	data.Init(chunkSize)
	lastIdx := len(data.bufs) - 1

	go func() {
		for i := 0; i < size/4; i++ {
			v := rand.New(rand.NewSource(int64(i))).Int63()
			data.bufs[lastIdx] = strconv.AppendInt(data.bufs[lastIdx], v, 10)

			if len(data.bufs[lastIdx]) <= chunkSize {
				data.mu.Lock()
				isWaitWrite := false
				if data.used == len(data.bufs[lastIdx]) {
					isWaitWrite = true
				}
				data.bufs = append(data.bufs, make([]byte, 0, chunkSize))
				lastIdx = len(data.bufs) - 1
				data.mu.Unlock()
				if isWaitWrite {
					data.wgw.Done()
				}
			}

			if len(data.bufs) == cap(data.bufs) {
				data.wgr.Add(1)
				data.wgr.Wait()
				lastIdx = 0
			}

		}
	}()
	return &data

}

func Test_CompressBuf(t *testing.T) {

	data := sampleData(1024*1024, 4098)
	err := compressing(data,
		func(lzw *lz4.Writer) {
			io.Copy(lzw, data)
		},
		func(fname string) bool {
			return true
		})
	assert.NoError(t, err)

}

func Benchmark_Compress(b *testing.B) {

	fn := func(b *testing.B) {
		data := sampleData(1024*b.N, 4098)
		b.ResetTimer()
		compressing(data,
			func(lzw *lz4.Writer) {
				io.Copy(lzw, data)
			},
			func(fname string) bool {
				return true
			})

	}
	b.Run("io.Copy", fn)
	b.Run("lzw.Write", func(b *testing.B) {
		data := sampleData(1024*b.N, 4098)
		b.ResetTimer()
		compressing(data,
			func(lzw *lz4.Writer) {

				for {
					lzw.Write(data.bufs[data.used])
					data.waitAndRound()
				}
			},
			func(fname string) bool {
				return true
			})

	})

}

func compressing(data *sampleDataInfo, writeFn func(lzw *lz4.Writer), onSucc func(fname string) bool) error {

	const (
		TmpCompFname = "../tmp/normal_comp.lz4"
	)

	f, err := os.Create(TmpCompFname)
	if err != nil {
		return err
	}
	lzw := lz4.NewWriter(f)
	defer lzw.Close()
	writeFn(lzw)
	err = lzw.Flush()
	if err != nil {
		return err
	}

	ok := onSucc(TmpCompFname)
	if ok {
		err = os.Remove(TmpCompFname)
	}
	if err != nil {
		return err
	}
	return nil
}

func Test_Compress(t *testing.T) {

	const (
		TmpCompFname = "../tmp/normal_comp.lz4"
	)

	data := makeSampleData(1024 * 1024)

	f, err := os.Create(TmpCompFname)
	assert.NoErrorf(t, err, "create file")
	if err != nil {
		return
	}

	v := true
	assert.Equal(t, true, v)

	lzw := lz4.NewWriter(f)
	lzw.Write(data)
	//lzw.ReadFrom(bytes.NewBuffer(data))

	err = lzw.Close()
	assert.NoErrorf(t, err, "close lz4 file")
	if err != nil {
		return
	}
	info, err := os.Stat(TmpCompFname)
	assert.NoErrorf(t, err, "stat lz4 file")
	if err != nil {
		return
	}
	assert.True(t, info.Size() > 0)

	err = os.Remove(TmpCompFname)
	assert.NoErrorf(t, err, "remove file")

}
