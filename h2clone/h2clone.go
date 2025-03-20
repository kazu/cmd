package h2clone

import (
	"os"
)

func CloneFile(src, dst string) error {
	clonepath := dst + ".clone"
	backuppath := dst + ".backup"
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	err = FileClose(srcFile, clonepath)
	if err != nil {
		return err
	}
	defer os.Remove(clonepath)

	err = os.Rename(dst, backuppath)
	if err != nil {
		return err
	}
	defer os.Remove(backuppath)

	err = os.Rename(clonepath, dst)
	return err

}
