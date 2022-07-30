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

package logger

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

func reset() {
	SetLogger(nil)
}

func TestInitLogger(t *testing.T) {
	config := Config{
		Level: "degug",
	}
	err := InitLogger(config)
	assert.NoError(t, err)
	reset()
}

func TestGetLogger(t *testing.T) {
	// not yet init get default log
	log := GetLogger()
	config := Config{
		Level: "degug",
	}
	_ = InitLogger(config)
	// after init logger
	log2 := GetLogger()
	assert.NotEqual(t, log, log2)

	// the secend init logger
	config.Level = "info"
	_ = InitLogger(config)
	log3 := GetLogger()
	assert.NotEqual(t, log2, log3)
	reset()
}

func TestSetLogger(t *testing.T) {
	// not yet init get default log
	log := GetLogger()
	log1 := &mockLogger{}
	SetLogger(log1)

	// after set logger
	log2 := GetLogger()
	assert.NotEqual(t, log, log2)
	assert.Equal(t, log1, log2)

	config := Config{
		Level: "degug",
	}
	_ = InitLogger(config)
	// after init logger
	log3 := GetLogger()
	assert.NotEqual(t, log2, log3)
	reset()
}

func TestRaceLogger(t *testing.T) {
	wg := sync.WaitGroup{}
	for i := 0; i < 100; i++ {
		wg.Add(3)
		go func() {
			defer wg.Done()
			SetLogger(&mockLogger{})
		}()
		go func() {
			defer wg.Done()
			_ = GetLogger()
		}()
		go func() {
			defer wg.Done()
			config := Config{
				Level: "degug",
			}
			_ = InitLogger(config)
		}()
	}
	wg.Wait()
	reset()
}

type mockLogger struct {
}

func (m mockLogger) Info(args ...interface{}) {
	panic("implement me")
}

func (m mockLogger) Warn(args ...interface{}) {
	panic("implement me")
}

func (m mockLogger) Error(args ...interface{}) {
	panic("implement me")
}

func (m mockLogger) Debug(args ...interface{}) {
	panic("implement me")
}

func (m mockLogger) Infof(fmt string, args ...interface{}) {
	panic("implement me")
}

func (m mockLogger) Warnf(fmt string, args ...interface{}) {
	panic("implement me")
}

func (m mockLogger) Errorf(fmt string, args ...interface{}) {
	panic("implement me")
}

func (m mockLogger) Debugf(fmt string, args ...interface{}) {
	panic("implement me")
}
