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
	"math"
	"reflect"
	"sync"
	"sync/atomic"
	"time"

	"github.com/pkg/errors"

	"github.com/nacos-group/nacos-sdk-go/v2/common/constant"
	"github.com/nacos-group/nacos-sdk-go/v2/common/logger"
	"github.com/nacos-group/nacos-sdk-go/v2/common/nacos_server"
	"github.com/nacos-group/nacos-sdk-go/v2/common/remote/rpc/rpc_request"
	"github.com/nacos-group/nacos-sdk-go/v2/common/remote/rpc/rpc_response"
	"github.com/nacos-group/nacos-sdk-go/v2/util"
)

type ConnectionType uint32

const (
	GRPC ConnectionType = iota
)

type RpcClientStatus int32

const (
	INITIALIZED RpcClientStatus = iota
	STARTING
	UNHEALTHY
	RUNNING
	SHUTDOWN
)

func (status RpcClientStatus) getDesc() string {
	switch status {
	case INITIALIZED:
		return "INITIALIZED"
	case STARTING:
		return "STARTING"
	case UNHEALTHY:
		return "UNHEALTHY"
	case RUNNING:
		return "RUNNING"
	case SHUTDOWN:
		return "SHUTDOWN"
	default:
		return "UNKNOWN"
	}
}

type ConnectionStatus uint32

const (
	DISCONNECTED ConnectionStatus = iota
	CONNECTED
)

var (
	cMux      = new(sync.Mutex)
	clientMap = make(map[string]IRpcClient)
)

type IRpcClient interface {
	connectToServer(serverInfo ServerInfo) (IConnection, error)
	getConnectionType() ConnectionType
	putAllLabels(labels map[string]string)
	rpcPortOffset() uint64
	GetRpcClient() *RpcClient
}

type ServerInfo struct {
	serverIp       string
	serverPort     uint64
	serverGrpcPort uint64
}

type RpcClient struct {
	Name                        string
	labels                      map[string]string
	currentConnection           IConnection
	rpcClientStatus             RpcClientStatus
	eventChan                   chan ConnectionEvent
	reconnectionChan            chan ReconnectContext
	connectionEventListeners    atomic.Value
	lastActiveTimestamp         atomic.Value
	executeClient               IRpcClient
	nacosServer                 *nacos_server.NacosServer
	serverRequestHandlerMapping map[string]ServerRequestHandlerMapping
	mux                         *sync.Mutex
	clientAbilities             rpc_request.ClientAbilities
	Tenant                      string
	stopChan                    chan struct{}
}

type ServerRequestHandlerMapping struct {
	serverRequest func() rpc_request.IRequest
	handler       IServerRequestHandler
}

type ReconnectContext struct {
	onRequestFail bool
	serverInfo    ServerInfo
}

type ConnectionEvent struct {
	eventType ConnectionStatus
}

func (r *RpcClient) putAllLabels(labels map[string]string) {
	for k, v := range labels {
		r.labels[k] = v
	}
}

func (r *RpcClient) GetRpcClient() *RpcClient {
	return r
}

/**
 * get all client.
 *
 */
func getAllClient() map[string]IRpcClient {
	return clientMap
}

func getClient(clientName string) IRpcClient {
	return clientMap[clientName]
}

func CreateClient(clientName string, connectionType ConnectionType, labels map[string]string, nacosServer *nacos_server.NacosServer) (IRpcClient, error) {
	cMux.Lock()
	defer cMux.Unlock()
	if _, ok := clientMap[clientName]; !ok {
		var rpcClient IRpcClient
		if GRPC == connectionType {
			rpcClient = NewGrpcClient(clientName, nacosServer)
		}
		if rpcClient == nil {
			return nil, errors.New("unsupported connection type")
		}
		rpcClient.putAllLabels(labels)
		clientMap[clientName] = rpcClient
		return rpcClient, nil
	}
	return clientMap[clientName], nil
}

func (r *RpcClient) Start() {
	if ok := atomic.CompareAndSwapInt32((*int32)(&r.rpcClientStatus), (int32)(INITIALIZED), (int32)(STARTING)); !ok {
		return
	}
	r.registerServerRequestHandlers()

	go func() {
		defer func() {
			logger.Infof("goroutine for notifying connection events of rpc client:%s exit", r.Name)
		}()

		for {
			event := <-r.eventChan
			r.notifyConnectionEvent(event)
			select {
			case <-r.stopChan:
				return
			default:
			}
		}
	}()

	go func() {
		defer func() {
			logger.Infof("goroutine for reconnecting to server of rpc client:%s exit", r.Name)
		}()

		timer := time.NewTimer(5 * time.Second)
		for {
			select {
			case rc := <-r.reconnectionChan:
				if (rc.serverInfo != ServerInfo{}) {
					var serverExist bool
					for _, v := range r.nacosServer.GetServerList() {
						if rc.serverInfo.serverIp == v.IpAddr {
							rc.serverInfo.serverPort = v.Port
							rc.serverInfo.serverGrpcPort = v.GrpcPort
							serverExist = true
							break
						}
					}
					if !serverExist {
						logger.Infof("%s recommend server is not in server list, ignore recommend server %+v", r.Name, rc.serverInfo)
						rc.serverInfo = ServerInfo{}
					}
				}
				r.reconnect(rc.serverInfo, rc.onRequestFail)
			case <-timer.C:
				r.healthCheck(timer)
			case <-r.nacosServer.ServerSrcChangeSignal:
				r.notifyServerSrvChange()
			case <-r.stopChan:
				timer.Stop()
				return
			}
		}
	}()

	var currentConnection IConnection
	startUpRetryTimes := constant.REQUEST_DOMAIN_RETRY_TIME
	for startUpRetryTimes > 0 && currentConnection == nil {
		startUpRetryTimes--
		serverInfo, err := r.nextRpcServer()
		if err != nil {
			logger.Errorf("[RpcClient.nextRpcServer],err:%+v", err)
			break
		}
		logger.Infof("[RpcClient.Start] %s try to connect to server on start up, server: %+v", r.Name, serverInfo)
		if connection, err := r.executeClient.connectToServer(serverInfo); err != nil {
			logger.Warnf("[RpcClient.Start] %s fail to connect to server on start up, error message=%v, "+
				"start up retry times left=%d", r.Name, err.Error(), startUpRetryTimes)
		} else {
			currentConnection = connection
			break
		}
	}
	if currentConnection != nil {
		logger.Infof("%s success to connect to server %+v on start up, connectionId=%s", r.Name,
			currentConnection.getServerInfo(), currentConnection.getConnectionId())
		r.currentConnection = currentConnection
		atomic.StoreInt32((*int32)(&r.rpcClientStatus), (int32)(RUNNING))
		r.eventChan <- ConnectionEvent{eventType: CONNECTED}
	} else {
		r.switchServerAsync(ServerInfo{}, false)
	}
}

func (r *RpcClient) notifyServerSrvChange() {
	if r.currentConnection == nil {
		r.switchServerAsync(ServerInfo{}, false)
		return
	}
	curServerInfo := r.currentConnection.getServerInfo()
	var found bool
	for _, ele := range r.nacosServer.GetServerList() {
		if ele.IpAddr == curServerInfo.serverIp {
			found = true
		}
	}
	if !found {
		logger.Infof("Current connected server %s:%d is not in latest server list, switch switchServerAsync", curServerInfo.serverIp, curServerInfo.serverPort)
		r.switchServerAsync(ServerInfo{}, false)
	}
}

func (r *RpcClient) registerServerRequestHandlers() {
	// register ConnectResetRequestHandler.
	r.RegisterServerRequestHandler(func() rpc_request.IRequest {
		return &rpc_request.ConnectResetRequest{InternalRequest: rpc_request.NewInternalRequest()}
	}, &ConnectResetRequestHandler{})

	// register client detection request.
	r.RegisterServerRequestHandler(func() rpc_request.IRequest {
		return &rpc_request.ClientDetectionRequest{InternalRequest: rpc_request.NewInternalRequest()}
	}, &ClientDetectionRequestHandler{})
}

func (r *RpcClient) Shutdown() {
	atomic.StoreInt32((*int32)(&r.rpcClientStatus), (int32)(SHUTDOWN))
	r.closeConnection()
	close(r.stopChan)
}

func (r *RpcClient) RegisterServerRequestHandler(request func() rpc_request.IRequest, handler IServerRequestHandler) {
	requestType := request().GetRequestType()
	if handler == nil || requestType == "" {
		logger.Errorf("%s register server push request handler "+
			"missing required parameters,request:%+v handler:%+v", r.Name, requestType, handler.Name())
		return
	}
	logger.Debugf("%s register server push request:%s handler:%+v", r.Name, requestType, handler.Name())
	r.serverRequestHandlerMapping[requestType] = ServerRequestHandlerMapping{
		serverRequest: request,
		handler:       handler,
	}
}

func (r *RpcClient) RegisterConnectionListener(listener IConnectionEventListener) {
	logger.Debugf("%s register connection listener [%+v] to current client", r.Name, reflect.TypeOf(listener))
	listeners := r.connectionEventListeners.Load()
	connectionEventListeners := listeners.([]IConnectionEventListener)
	connectionEventListeners = append(connectionEventListeners, listener)
	r.connectionEventListeners.Store(connectionEventListeners)
}

func (r *RpcClient) switchServerAsync(recommendServerInfo ServerInfo, onRequestFail bool) {
	r.reconnectionChan <- ReconnectContext{serverInfo: recommendServerInfo, onRequestFail: onRequestFail}
}

func (r *RpcClient) reconnect(serverInfo ServerInfo, onRequestFail bool) {
	if onRequestFail && r.sendHealthCheck() {
		logger.Infof("%s server check success, currentServer is %+v", r.Name, r.currentConnection.getServerInfo())
		atomic.StoreInt32((*int32)(&r.rpcClientStatus), (int32)(RUNNING))
		return
	}
	var (
		serverInfoFlag             bool
		reConnectTimes, retryTurns int
		err                        error
	)
	if (serverInfo == ServerInfo{}) {
		serverInfoFlag = true
		logger.Infof("%s try to re connect to a new server, server is not appointed, will choose a random server.", r.Name)
	}

	for !r.isShutdown() {
		if serverInfoFlag {
			serverInfo, err = r.nextRpcServer()
			if err != nil {
				logger.Errorf("[RpcClient.nextRpcServer],err:%v", err)
				break
			}
		}
		connectionNew, err := r.executeClient.connectToServer(serverInfo)
		if connectionNew != nil && err == nil {
			logger.Infof("%s success to connect a server %+v, connectionId=%s", r.Name, serverInfo,
				connectionNew.getConnectionId())

			if r.currentConnection != nil {
				logger.Infof("%s abandon prev connection, server is %+v, connectionId is %s", r.Name, serverInfo,
					r.currentConnection.getConnectionId())
				r.currentConnection.setAbandon(true)
				r.closeConnection()
			}
			r.currentConnection = connectionNew
			atomic.StoreInt32((*int32)(&r.rpcClientStatus), (int32)(RUNNING))
			r.eventChan <- ConnectionEvent{eventType: CONNECTED}
			return
		}
		if r.isShutdown() {
			r.closeConnection()
		}
		if reConnectTimes > 0 && reConnectTimes%len(r.nacosServer.GetServerList()) == 0 {
			logger.Warnf("%s fail to connect server, after trying %d times, last try server is %+v, error=%v", r.Name,
				reConnectTimes, serverInfo, err)
			if retryTurns < 50 {
				retryTurns++
			}
		}
		reConnectTimes++
		if !r.IsRunning() {
			time.Sleep(time.Duration((math.Min(float64(retryTurns), 50))*100) * time.Millisecond)
		}
	}
	if r.isShutdown() {
		logger.Warnf("%s client is shutdown, stop reconnect to server", r.Name)
	}
}

func (r *RpcClient) closeConnection() {
	if r.currentConnection != nil {
		r.currentConnection.close()
		r.eventChan <- ConnectionEvent{eventType: DISCONNECTED}
	}
}

// Notify when client new connected.
func (r *RpcClient) notifyConnectionEvent(event ConnectionEvent) {
	listeners := r.connectionEventListeners.Load().([]IConnectionEventListener)
	if len(listeners) == 0 {
		return
	}
	logger.Infof("%s notify %s event to listeners.", r.Name, event.toString())
	for _, v := range listeners {
		if event.isConnected() {
			v.OnConnected()
		}
		if event.isDisConnected() {
			v.OnDisConnect()
		}
	}
}

func (r *RpcClient) healthCheck(timer *time.Timer) {
	defer timer.Reset(constant.KEEP_ALIVE_TIME * time.Second)
	var reconnectContext ReconnectContext
	lastActiveTimeStamp := r.lastActiveTimestamp.Load().(time.Time)
	if time.Now().Sub(lastActiveTimeStamp) < constant.KEEP_ALIVE_TIME*time.Second {
		return
	}
	if r.sendHealthCheck() {
		r.lastActiveTimestamp.Store(time.Now())
		return
	} else {
		if r.currentConnection == nil {
			return
		}
		logger.Infof("%s server healthy check fail, currentConnection=%s", r.Name, r.currentConnection.getConnectionId())
		atomic.StoreInt32((*int32)(&r.rpcClientStatus), (int32)(UNHEALTHY))
		reconnectContext = ReconnectContext{onRequestFail: false}
	}
	r.reconnect(reconnectContext.serverInfo, reconnectContext.onRequestFail)
}

func (r *RpcClient) sendHealthCheck() bool {
	if r.currentConnection == nil {
		return false
	}
	response, err := r.currentConnection.request(rpc_request.NewHealthCheckRequest(),
		constant.DEFAULT_TIMEOUT_MILLS, r)
	if err != nil {
		return false
	}
	if !response.IsSuccess() {
		// when client request immediately after the nacos server starts, the server may not ready to serve new request
		// the server will return code 3xx, tell the client to retry after a while
		// this situation, just return true,because the healthCheck will start again after 5 seconds
		if response.GetErrorCode() >= 300 && response.GetErrorCode() < 400 {
			return true
		}
		return false
	}
	return true
}

func (r *RpcClient) nextRpcServer() (ServerInfo, error) {
	serverConfig, err := r.nacosServer.GetNextServer()
	if err != nil {
		return ServerInfo{}, err
	}
	return ServerInfo{
		serverIp:       serverConfig.IpAddr,
		serverPort:     serverConfig.Port,
		serverGrpcPort: serverConfig.GrpcPort,
	}, nil
}

func (c *ConnectionEvent) isConnected() bool {
	return c.eventType == CONNECTED
}

func (c *ConnectionEvent) isDisConnected() bool {
	return c.eventType == DISCONNECTED
}

//check is this client is shutdown.
func (r *RpcClient) isShutdown() bool {
	return atomic.LoadInt32((*int32)(&r.rpcClientStatus)) == (int32)(SHUTDOWN)
}

//IsRunning check is this client is running.
func (r *RpcClient) IsRunning() bool {
	return atomic.LoadInt32((*int32)(&r.rpcClientStatus)) == (int32)(RUNNING)
}

func (r *RpcClient) IsInitialized() bool {
	return atomic.LoadInt32((*int32)(&r.rpcClientStatus)) == (int32)(INITIALIZED)
}

func (c *ConnectionEvent) toString() string {
	if c.isConnected() {
		return "connected"
	}
	if c.isDisConnected() {
		return "disconnected"
	}
	return ""
}

func (r *RpcClient) Request(request rpc_request.IRequest, timeoutMills int64) (rpc_response.IResponse, error) {
	retryTimes := 0
	start := util.CurrentMillis()
	var currentErr error
	for retryTimes < constant.REQUEST_DOMAIN_RETRY_TIME && util.CurrentMillis() < start+timeoutMills {
		if r.currentConnection == nil || !r.IsRunning() {
			currentErr = waitReconnect(timeoutMills, &retryTimes, request,
				errors.Errorf("client not connected, current status:%s", r.rpcClientStatus.getDesc()))
			continue
		}
		response, err := r.currentConnection.request(request, timeoutMills, r)
		if err == nil {
			if response, ok := response.(*rpc_response.ErrorResponse); ok {
				if response.GetErrorCode() == constant.UN_REGISTER {
					r.mux.Lock()
					if atomic.CompareAndSwapInt32((*int32)(&r.rpcClientStatus), (int32)(RUNNING), (int32)(UNHEALTHY)) {
						logger.Infof("Connection is unregistered, switch server, connectionId=%s, request=%s",
							r.currentConnection.getConnectionId(), request.GetRequestType())
						r.switchServerAsync(ServerInfo{}, false)
					}
					r.mux.Unlock()
				}
				currentErr = waitReconnect(timeoutMills, &retryTimes, request, errors.New(response.GetMessage()))
				continue
			}
			r.lastActiveTimestamp.Store(time.Now())
			return response, nil
		} else {
			currentErr = waitReconnect(timeoutMills, &retryTimes, request, err)
		}
	}

	if atomic.CompareAndSwapInt32((*int32)(&r.rpcClientStatus), int32(RUNNING), int32(UNHEALTHY)) {
		r.switchServerAsync(ServerInfo{}, true)
	}
	if currentErr != nil {
		return nil, currentErr
	}
	return nil, errors.New("request fail, unknown error")
}

func waitReconnect(timeoutMills int64, retryTimes *int, request rpc_request.IRequest, err error) error {
	logger.Errorf("Send request fail, request=%s, body=%s, retryTimes=%v, error=%+v", request.GetRequestType(), request.GetBody(request), *retryTimes, err)
	time.Sleep(time.Duration(math.Min(100, float64(timeoutMills/3))) * time.Millisecond)
	*retryTimes++
	return err
}
