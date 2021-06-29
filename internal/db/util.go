package db

import (
	"hash/crc32"
	"io"
	"os"
	"path/filepath"
	"strings"
)

func roundDown(total, factor int64) int64 {
	return factor * (total / factor)
}

// CopyFile CopyFile
func CopyFile(dstName, srcName string) (written int64, err error) {
	src, err := os.Open(srcName)
	if err != nil {
		return
	}
	defer src.Close()
	dst, err := os.OpenFile(dstName, os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		return
	}
	defer dst.Close()
	return io.Copy(dst, src)
}

// GetSlotNum GetSlotNum
func GetSlotNum(slotCount int, v string) uint {
	value := crc32.ChecksumIEEE([]byte(v))
	return uint(value % uint32(slotCount))
}

// GetDirList GetDirList
func GetDirList(dirpath string) ([]string, error) {
	var dirList []string
	dirErr := filepath.Walk(dirpath,
		func(path string, f os.FileInfo, err error) error {
			if f == nil {
				return err
			}
			if f.IsDir() {
				dirList = append(dirList, path)
				return nil
			}

			return nil
		})
	return dirList, dirErr
}

// GetFileList GetFileList
func GetFileList(dirpath string, suffix string) ([]string, error) {
	var resultList []string
	dirErr := filepath.Walk(dirpath,
		func(path string, f os.FileInfo, err error) error {
			if f == nil {
				return err
			}
			if !f.IsDir() {
				if strings.HasSuffix(path, suffix) {
					resultList = append(resultList, path)
				}
				return nil
			}

			return nil
		})
	return resultList, dirErr
}
