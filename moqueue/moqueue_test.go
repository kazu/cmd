package moqueue_test

import (
	"crypto/sha1"
	"os"
	"sync"
	"testing"
	"unsafe"

	"github.com/stretchr/testify/assert"
)

func makeRandomData(size int) (filename string) {

	f, err := os.CreateTemp("", "randam.*.data")
	if err != nil {
		return ""
	}
	filename = f.Name()
	var tmp []byte
	for i := 0; i < size; {

		h := sha1.Sum(nil)
		tmp = unsafe.Slice(&h[0], 20)
		f.Write(tmp)
		i += len(tmp)
	}

	if err := f.Close(); err != nil {
		os.Remove(filename)
		return ""
	}
	return filename

}

func Test_MultiRead(t *testing.T) {

	fname := makeRandomData(100)

	assert.True(t, len(fname) > 0)
	defer os.Remove(fname)

	info, err := os.Stat(fname)

	assert.NoError(t, err)

	assert.Equal(t, int64(100), info.Size())

}

func Benchmark_MultiRead(b *testing.B) {

	fsize := 4096 * 4096
	//b.N = 10

	fn := func(fname string, start, end int64, wg *sync.WaitGroup) {

		f, _ := os.Open(fname)

		tmp := make([]byte, 4096)
		for reads := int64(0); reads < end-start; {
			n, err := f.ReadAt(tmp, reads)
			if err != nil {
				break
			}
			reads += int64(n)

		}
		f.Close()
		wg.Done()
	}

	b.ResetTimer()
	b.Run("normal", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			b.StopTimer()
			fname := makeRandomData(fsize)

			var wg sync.WaitGroup

			wg.Add(1)

			b.StartTimer()
			go fn(fname, 0, int64(fsize), &wg)
			wg.Wait()

			b.StopTimer()
			os.Remove(fname)
		}
	})

	b.ResetTimer()
	b.Run("2      ", func(b *testing.B) {

		for i := 0; i < b.N; i++ {
			b.StopTimer()
			fname := makeRandomData(fsize)
			var wg sync.WaitGroup

			wg.Add(1)
			wg.Add(1)
			b.StartTimer()

			go fn(fname, 0, int64(fsize/2), &wg)
			go fn(fname, int64(fsize/2), int64(fsize), &wg)
			wg.Wait()

			b.StopTimer()
			os.Remove(fname)

		}
	})

}
