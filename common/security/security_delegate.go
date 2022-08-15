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
	"sync/atomic"

	"github.com/nacos-group/nacos-sdk-go/v2/common/logger"

	"github.com/nacos-group/nacos-sdk-go/v2/common/constant"

	"github.com/nacos-group/nacos-sdk-go/v2/model"
)

type AuthClientDelegate struct {
	authServices []model.ClientAuthService
	status       int32
}

// NewAuthClientDelegate create new AuthDelegate with authService
func NewAuthClientDelegate(authServices []model.ClientAuthService) *AuthClientDelegate {
	if len(authServices) == 0 {
		logger.Warn("there is no client auth service using to create auth client delegate")
	}
	return &AuthClientDelegate{
		authServices: authServices,
	}
}

// Login to login all auth services
func (manager *AuthClientDelegate) Login() error {
	size := len(manager.authServices)
	if size == 0 {
		logger.Warn("there is no client auth service")
		return nil
	}
	for i := 0; i < size; i++ {
		service := manager.authServices[i]
		login, err := service.Login()
		if !login {
			logger.Error(fmt.Sprintf("%s had logined fail.becase of %v", service.Name(), err))
		}
		if err != nil {
			return err
		}
	}
	return nil
}

// GetAllContent collect all auth service content
func (manager *AuthClientDelegate) GetAllContent() map[string]string {
	result := make(map[string]string)
	for _, as := range manager.authServices {
		content := as.GetLoginContent()
		if len(content) > 0 {
			for k, v := range content {
				if _, ok := result[k]; ok {
					logger.Warn(fmt.Sprintf("some auth service contain the same content key!the same key is %s", k))
				}
				result[k] = v
			}
		}
	}
	return result
}

// AddNewAuthService add new auth service to manager
func (manager *AuthClientDelegate) AddNewAuthService(service model.ClientAuthService) {
	manager.authServices = append(manager.authServices, service)
	if manager.status == constant.AUTH_MANAGER_STARTED {
		// if auth client has start refresh,the new ClientAuthService should start auto refresh too.
		go func() {
			service.AutoRefresh()
		}()
	}
}

// StartAutoRefresh start fresh auth service,this func won't spin to refresh content, it only trigger all auth service's auth refresh func.
func (manager *AuthClientDelegate) StartAutoRefresh() {
	// ensure refresh only once.
	if atomic.LoadInt32(&manager.status) == constant.AUTH_MANAGER_STARTED {
		return
	}
	atomic.StoreInt32(&manager.status, constant.AUTH_MANAGER_STARTED)
	go func() {
		for _, as := range manager.authServices {
			as.AutoRefresh()
		}
	}()
}
