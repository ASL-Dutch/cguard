package utils

import (
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
)

// IsDir Path is directory
func IsDir(fileAddr string) bool {
	s, err := os.Stat(fileAddr)
	if err != nil {
		log.Println(err)
		return false
	}
	return s.IsDir()
}

// CreateDir creates a directory
func CreateDir(dir string) bool {
	err := os.MkdirAll(dir, os.ModePerm)
	if err != nil {
		log.Println(err)
		return false
	}
	return true
}

// IsExists Path is exists
func IsExists(path string) bool {
	_, err := os.Stat(path)
	if err == nil {
		return true
	}
	return false
}

// Copy Path to Path
func Copy(srcFile, dstFile string) error {
	srcStat, err := os.Stat(srcFile)
	if err != nil {
		return err
	}

	if !srcStat.Mode().IsRegular() {
		return fmt.Errorf("%s is not a regular file", srcFile)
	}

	source, err := os.Open(srcFile)
	if err != nil {
		return err
	}
	defer source.Close()

	dest, err := os.Create(dstFile)
	if err != nil {
		return err
	}
	defer dest.Close()

	_, err = io.Copy(dest, source)
	if err != nil {
		return err
	}
	return nil
}

const DefaultTimeLayout = "20060102150405"

// GetFilePath 根据filename获取文件路径
// 如：filename为OP210603005_20220909153131.xlsx,
// 返回：{rootDir}/2022/6/OP210603005_20220909153131.xlsx
func GetFilePathByFilename(rootDir, filename, timeLayout string) (string, error) {
	if timeLayout == "" {
		timeLayout = DefaultTimeLayout
	}

	fn := strings.Split(filename, ".")[0]
	paths := strings.Split(fn, "_")
	timestamp := paths[len(paths)-1]
	if timestamp == "" {
		return "", fmt.Errorf("the filename: %s cannot get timestamp", filename)
	}
	ftime, err := time.Parse(timeLayout, timestamp)
	if err != nil {
		return "", fmt.Errorf("the timestamp: %s is not valid(timeLayout: %s)", timestamp, timeLayout)
	}
	filepath := fmt.Sprintf("%s/%d/%d/%s", rootDir, ftime.Year(), ftime.Month(), filename)

	return filepath, nil
}

// CopyFile 复制文件
func CopyFile(src, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	dstFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer dstFile.Close()

	_, err = io.Copy(dstFile, srcFile)
	if err != nil {
		return err
	}
	return nil
}

// RenameFile 重命名文件
func RenameFile(oldPath, newPath string) error {
	return os.Rename(oldPath, newPath)
}
