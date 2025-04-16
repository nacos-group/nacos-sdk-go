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
	"context"
	"time"

	"github.com/dbsyk/nacos-sdk-go/v2/common/logger"

	nacos_grpc_service "github.com/dbsyk/nacos-sdk-go/v2/api/grpc"
	"github.com/dbsyk/nacos-sdk-go/v2/common/remote/rpc/rpc_request"
	"github.com/dbsyk/nacos-sdk-go/v2/common/remote/rpc/rpc_response"
	"github.com/dbsyk/nacos-sdk-go/v2/util"
	"github.com/golang/protobuf/ptypes/any"
	"github.com/pkg/errors"

	"google.golang.org/grpc"
)

type GrpcConnection struct {
	*Connection
	client         nacos_grpc_service.RequestClient
	biStreamClient nacos_grpc_service.BiRequestStream_RequestBiStreamClient
}

func NewGrpcConnection(serverInfo ServerInfo, connectionId string, conn *grpc.ClientConn,
	client nacos_grpc_service.RequestClient, biStreamClient nacos_grpc_service.BiRequestStream_RequestBiStreamClient) *GrpcConnection {
	return &GrpcConnection{
		Connection: &Connection{
			serverInfo:   serverInfo,
			connectionId: connectionId,
			abandon:      false,
			conn:         conn,
		},
		client:         client,
		biStreamClient: biStreamClient,
	}
}
func (g *GrpcConnection) request(request rpc_request.IRequest, timeoutMills int64, client *RpcClient) (rpc_response.IResponse, error) {
	p := convertRequest(request)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeoutMills)*time.Millisecond)
	defer cancel()
	responsePayload, err := g.client.Request(ctx, p)
	if err != nil {
		logger.Debugf("%s grpc request nacos server failed, request=%+v, err=%v ", g.getConnectionId(), p, err)
		return nil, err
	}

	responseFunc, ok := rpc_response.ClientResponseMapping[responsePayload.Metadata.GetType()]
	if !ok {
		return nil, errors.Errorf("request:%s,unsupported response type:%s", request.GetRequestType(),
			responsePayload.Metadata.GetType())
	}

	logger.Debugf("%s grpc request nacos server success, request=%+v, response=%s", g.getConnectionId(), p, string(responsePayload.GetBody().Value))
	return rpc_response.InnerResponseJsonUnmarshal(responsePayload.GetBody().Value, responseFunc)
}

func (g *GrpcConnection) close() {
	g.Connection.close()
}

func (g *GrpcConnection) biStreamSend(payload *nacos_grpc_service.Payload) error {
	return g.biStreamClient.Send(payload)
}

func convertRequest(r rpc_request.IRequest) *nacos_grpc_service.Payload {
	Metadata := nacos_grpc_service.Metadata{
		Type:     r.GetRequestType(),
		Headers:  r.GetHeaders(),
		ClientIp: util.LocalIP(),
	}
	return &nacos_grpc_service.Payload{
		Metadata: &Metadata,
		Body:     &any.Any{Value: []byte(r.GetBody(r))},
	}
}

func convertResponse(r rpc_response.IResponse) *nacos_grpc_service.Payload {
	Metadata := nacos_grpc_service.Metadata{
		Type:     r.GetResponseType(),
		ClientIp: util.LocalIP(),
	}
	return &nacos_grpc_service.Payload{
		Metadata: &Metadata,
		Body:     &any.Any{Value: []byte(r.GetBody())},
	}
}
