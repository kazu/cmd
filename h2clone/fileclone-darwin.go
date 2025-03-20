//go:build darwin

package h2clone

import (
	"os"

	"golang.org/x/sys/unix"
)

func FileClose(srcFile *os.File, dst string) error {
	return unix.Clonefile(srcFile.Name(), dst, 0)
	//return unix.IoctlFileClone(int(dstFile.Fd()), int(srcFile.Fd()))
}
