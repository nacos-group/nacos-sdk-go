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

package constant

func NewServerConfig(ipAddr string, port uint64, opts ...ServerOption) *ServerConfig {
	serverConfig := &ServerConfig{
		IpAddr:      ipAddr,
		Port:        port,
		ContextPath: DEFAULT_CONTEXT_PATH,
		Scheme:      DEFAULT_SERVER_SCHEME,
	}

	for _, opt := range opts {
		opt(serverConfig)
	}

	return serverConfig
}

// ServerOption ...
type ServerOption func(*ServerConfig)

//WithScheme set Scheme for server
func WithScheme(scheme string) ServerOption {
	return func(config *ServerConfig) {
		config.Scheme = scheme
	}
}

//WithContextPath set contextPath for server
func WithContextPath(contextPath string) ServerOption {
	return func(config *ServerConfig) {
		config.ContextPath = contextPath
	}
}

//WithIpAddr set ip address for server
func WithIpAddr(ipAddr string) ServerOption {
	return func(config *ServerConfig) {
		config.IpAddr = ipAddr
	}
}

//WithPort set port for server
func WithPort(port uint64) ServerOption {
	return func(config *ServerConfig) {
		config.Port = port
	}
}

//WithGrpcPort set grpc port for server
func WithGrpcPort(port uint64) ServerOption {
	return func(config *ServerConfig) {
		config.GrpcPort = port
	}
}
