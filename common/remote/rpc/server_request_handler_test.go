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

package rpc

import (
	"github.com/nacos-group/nacos-sdk-go/v2/common/remote/rpc/rpc_request"
	"github.com/nacos-group/nacos-sdk-go/v2/common/remote/rpc/rpc_response"
	"sync"
	"testing"

	. "github.com/agiledragon/gomonkey/v2"
	. "github.com/smartystreets/goconvey/convey"
)

func TestConnectResetRequestHandler(t *testing.T) {
	Convey("expect return nil when request is not ConnectionResetRequest", t, func() {
		handler := &ConnectResetRequestHandler{}
		So(handler.RequestReply(nil, nil), ShouldBeNil)
	})

	Convey("expect return Connection Response", t, func() {
		handler := &ConnectResetRequestHandler{}
		req := &rpc_request.ConnectResetRequest{}
		client := &RpcClient{}
		client.mux = &sync.Mutex{}

		So(handler.RequestReply(req, client), ShouldHaveSameTypeAs, &rpc_response.ConnectResetResponse{})
	})

	Convey("expect call switchServerAsync with onRequestFail false when reply ConnectResetReq", t, func() {
		req := &rpc_request.ConnectResetRequest{}
		handler := &ConnectResetRequestHandler{}
		client := &RpcClient{}
		client.mux = &sync.Mutex{}
		flag := true
		patches := ApplyPrivateMethod(client, "switchServerAsync", func(recommendServerInfo ServerInfo, onRequestFail bool) {
			flag = onRequestFail
		})
		defer patches.Reset()
		patches.ApplyMethodReturn(client, "IsRunning", true)
		handler.RequestReply(req, client)
		So(flag, ShouldBeFalse)

		req.ServerPort = "1234"

		handler.RequestReply(req, client)
		So(flag, ShouldBeFalse)
	})

}
