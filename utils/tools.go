package utils

import (
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
)

// LocalPath 设置目录
func LocalPath(path string) (newPath string) {
	workDir, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}
	return filepath.Join(workDir, path)
}

// ReadFile 打开文件
func ReadFile(f string) string {
	data, err := ioutil.ReadFile(f)
	if err != nil {
		log.Fatal(err)
	}
	return string(data)
}
