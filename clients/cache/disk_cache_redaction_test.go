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

package cache

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"testing"
)

const sampleCacheValue = "test-only-secret-value"
const sampleCacheKey = "test-data@@DEFAULT_GROUP"

func TestCacheWriteFailureDoesNotExposeContent(t *testing.T) {
	for _, tc := range []struct {
		name string
		path func(string, string) string
	}{
		{"config", GetFileName},
		{"encrypted-data-key", GetConfigEncryptedDataKeyFileName},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			path := tc.path(sampleCacheKey, dir)
			if err := os.MkdirAll(path, 0700); err != nil {
				t.Fatal(err)
			}
			cmd := exec.Command(os.Args[0], "-test.run=^TestCacheWriteFailureHelper$", "-test.v")
			cmd.Env = append(os.Environ(), "NACOS_CACHE_TEST_CASE="+tc.name, "NACOS_CACHE_TEST_DIR="+dir)
			output, err := cmd.CombinedOutput()
			if err != nil {
				t.Fatalf("helper failed: %v\n%s", err, output)
			}
			text := string(output)
			if strings.Contains(text, sampleCacheValue) {
				t.Errorf("error or log contains the sample value: %s", text)
			}
			var logged, returned bool
			for _, line := range strings.Split(text, "\n") {
				if strings.Contains(line, "failed to write") && strings.Contains(line, path) {
					if strings.HasPrefix(line, "returned error:") {
						returned = true
					} else {
						logged = true
					}
				}
			}
			if !logged || !returned {
				t.Errorf("expected both a log and a returned error with the file path: %s", text)
			}
		})
	}
}

func TestCacheWriteFailureHelper(t *testing.T) {
	testCase := os.Getenv("NACOS_CACHE_TEST_CASE")
	if testCase == "" {
		t.Skip("subprocess helper")
	}
	dir := os.Getenv("NACOS_CACHE_TEST_DIR")
	var err error
	switch testCase {
	case "config":
		err = WriteConfigToFile(sampleCacheKey, dir, sampleCacheValue)
	case "encrypted-data-key":
		err = WriteEncryptedDataKeyToFile(sampleCacheKey, dir, sampleCacheValue)
	default:
		t.Fatalf("unknown test case: %s", testCase)
	}
	if err == nil {
		t.Fatal("expected writing to a directory to fail")
	}
	fmt.Printf("returned error: %v\n", err)
}
