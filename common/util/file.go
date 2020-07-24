package util

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

var osType string
var path string

const WINDOWS = "windows"

func init() {
	osType = runtime.GOOS
	if os.IsPathSeparator('\\') { //前边的判断是否是系统的分隔符
		path = "\\"
	} else {
		path = "/"
	}
}

func MkdirIfNecessary(createDir string) (err error) {
	s := strings.Split(createDir, path)
	startIndex := 0
	dir := ""
	if s[0] == "" {
		startIndex = 1
	} else {
		dir, _ = os.Getwd() //当前的目录
	}
	for i := startIndex; i < len(s); i++ {
		var d string
		if osType == WINDOWS && filepath.IsAbs(createDir) {
			d = strings.Join(s[startIndex:i+1], path)
		} else {
			d = dir + path + strings.Join(s[startIndex:i+1], path)
		}
		if _, e := os.Stat(d); os.IsNotExist(e) {
			err = os.Mkdir(d, os.ModePerm) //在当前目录下生成md目录
			if err != nil {
				break
			}
		}
	}

	return err
}
