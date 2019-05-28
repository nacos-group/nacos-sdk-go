// Copyright 2019 alibaba cloud
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Since: v0.1.0
// Author: github.com/atlanssia
package configs

import (
	"fmt"
	"io/ioutil"
	"math/rand"
	"net/http"
	"strings"
)

func GetConfigAddr(url string) (string, error) {
	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	v, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	addrs := strings.Split(strings.TrimSpace(string(v)), "\n")
	if addrs == nil || len(addrs) == 0 {
		return "", fmt.Errorf("no addr found: %+v", string(v))
	}
	return addrs[rand.Intn(len(addrs))], nil
}
