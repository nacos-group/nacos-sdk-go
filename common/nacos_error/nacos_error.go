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

package nacos_error

import (
	"fmt"

	"github.com/nacos-group/nacos-sdk-go/v2/common/constant"
)

const (
	/*
	 * client error code.
	 * -400 -503 throw exception to user.
	 */
	CLIENT_INVALID_PARAM  = -400 //invalid param（参数错误）
	CLIENT_DISCONNECT     = -401 //client disconnect.
	CLIENT_OVER_THRESHOLD = -503 //over client threshold（超过client端的限流阈值）.
	RESOURCE_NOT_FOUND    = -404
	CLIENT_ERROR          = -500 //client error（client异常，返回给服务端）.

	/*
	 * server error code.
	 * 400 403 throw exception to user
	 * 500 502 503 change ip and retry
	 */
	INVALID_PARAM         = 400 //invalid param（参数错误）.
	NO_RIGHT              = 403 //no right（鉴权失败）.
	NOT_FOUND             = 404 //not found.
	CONFLICT              = 409 //conflict（写并发冲突）.
	SERVER_ERROR          = 500 //server error（server异常，如超时）.
	BAD_GATEWAY           = 502 //bad gateway（路由异常，如nginx后面的Server挂掉）.
	OVER_THRESHOLD        = 503 //over threshold（超过server端的限流阈值）.
	INVALID_SERVER_STATUS = 300 //Server is not started.
	UN_REGISTER           = 301 //Connection is not registered.
	NO_HANDLER            = 302 //No Handler Found.

)

type NacosError struct {
	errorCode   string
	errMsg      string
	originError error
}

func NewNacosError(errorCode string, errMsg string, originError error) *NacosError {
	return &NacosError{
		errorCode:   errorCode,
		errMsg:      errMsg,
		originError: originError,
	}

}

func (err *NacosError) Error() (str string) {
	nacosErrMsg := fmt.Sprintf("[%s] %s", err.ErrorCode(), err.errMsg)
	if err.originError != nil {
		return nacosErrMsg + "\ncaused by:\n" + err.originError.Error()
	}
	return nacosErrMsg
}

func (err *NacosError) ErrorCode() string {
	if err.errorCode == "" {
		return constant.DefaultClientErrorCode
	} else {
		return err.errorCode
	}
}
