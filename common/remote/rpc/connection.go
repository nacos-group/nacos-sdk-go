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
	"google.golang.org/grpc"
)

type IConnection interface {
	request(request rpc_request.IRequest, timeoutMills int64, client *RpcClient) (rpc_response.IResponse, error)
	close()
	getConnectionId() string
	getServerInfo() ServerInfo
	setAbandon(flag bool)
	getAbandon() bool
}

type Connection struct {
	conn         *grpc.ClientConn
	connectionId string
	abandon      bool
	serverInfo   ServerInfo
}

func (c *Connection) getConnectionId() string {
	return c.connectionId
}

func (c *Connection) getServerInfo() ServerInfo {
	return c.serverInfo
}

func (c *Connection) setAbandon(flag bool) {
	c.abandon = flag
}

func (c *Connection) getAbandon() bool {
	return c.abandon
}

func (c *Connection) close() {
	_ = c.conn.Close()
}
