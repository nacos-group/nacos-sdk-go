/*
 * Copyright 1999-2020 Alibaba Group Holding Ltd.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *      http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package file

import (
	"log"
	"os"
	"path/filepath"
	"runtime"
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
	return os.MkdirAll(createDir, os.ModePerm)
}

func GetCurrentPath() string {
	dir, err := os.Getwd() //当前的目录
	if err != nil {
		dir, err = filepath.Abs(filepath.Dir(os.Args[0]))
		if err != nil {
			log.Println("can not get current path")
		}
	}
	return dir
}
