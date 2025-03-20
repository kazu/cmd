//go:build linux

package h2clone

import (
	"os"

	"golang.org/x/sys/unix"
)

func FileClose(srcFile *os.File, dst string) error {

	dstFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer dstFile.Close()

	err = unix.IoctlFileClone(int(dstFile.Fd()), int(srcFile.Fd()))
	if err != nil {
		os.Remove(dst)
	}
	return err
}
