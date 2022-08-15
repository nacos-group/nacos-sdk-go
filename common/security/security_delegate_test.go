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

package security

import (
	"fmt"
	"strconv"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/nacos-group/nacos-sdk-go/v2/common/logger"

	"github.com/nacos-group/nacos-sdk-go/v2/model"
)

// TestAuthClientManager create and use auth client manager
func TestAuthClientManager(t *testing.T) {
	// create new auth manager
	macs := &MockAuthClientService{name: "first"}
	service := []model.ClientAuthService{macs}
	acm := NewAuthClientDelegate(service)

	// login all client auth service
	err := acm.Login()
	assert.Nil(t, err)

	// get all auth content
	content := acm.GetAllContent()
	assert.NotNil(t, content)
	assert.Equal(t, 1, len(content))
	assert.Equal(t, content["count"], "1")

	// start auto refresh
	acm.StartAutoRefresh()
	time.Sleep(time.Second * 5)

	// assert the result will be the correct after sleep
	content2 := acm.GetAllContent()
	assert.True(t, content2["count"] == "5")

	// add new auth service
	newMacs := &MockAuthClientService{count: 10, name: "second"}
	acm.AddNewAuthService(newMacs)
	time.Sleep(time.Second * 1)
	// check if the content will be covered when content key is same.
	content3 := acm.GetAllContent()
	assert.NotNil(t, content3)
	assert.Equal(t, 1, len(content3))
	assert.Equal(t, content3["count"], "10")
}

// MockAuthClientService mock for test
type MockAuthClientService struct {
	count int32
	name  string
}

func (macs *MockAuthClientService) Name() string {
	return macs.name
}

func (macs *MockAuthClientService) Login() (bool, error) {
	logger.Info("start login with mock auth client")
	atomic.AddInt32(&macs.count, 1)
	return true, nil
}

func (macs *MockAuthClientService) GetLoginContent() model.LoginContent {
	r := atomic.LoadInt32(&macs.count)
	c := strconv.Itoa(int(r))
	return map[string]string{
		"count": c,
	}
}

func (macs *MockAuthClientService) AutoRefresh() {
	go func() {
		for {
			time.Sleep(time.Second)
			logger.Info(fmt.Sprintf("start refresh:%s", macs.Name()))
			macs.Login()
		}
	}()
}
