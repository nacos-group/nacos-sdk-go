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

import "time"

type ServerConfig struct {
	Scheme      string // the nacos server scheme,default=http,this is not required in 2.0
	ContextPath string // the nacos server contextpath,default=/nacos,this is not required in 2.0
	IpAddr      string // the nacos server address
	Port        uint64 // nacos server port
	GrpcPort    uint64 // nacos server grpc port, default=server port + 1000, this is not required
}

type ClientConfig struct {
	TimeoutMs            uint64                   // timeout for requesting Nacos server, default value is 10000ms
	ListenInterval       uint64                   // Deprecated
	BeatInterval         int64                    // the time interval for sending beat to server,default value is 5000ms
	NamespaceId          string                   // the namespaceId of Nacos.When namespace is public, fill in the blank string here.
	AppName              string                   // the appName
	AppKey               string                   // the client identity information
	Endpoint             string                   // the endpoint for get Nacos server addresses
	RegionId             string                   // the regionId for kms
	AccessKey            string                   // the AccessKey for kms
	SecretKey            string                   // the SecretKey for kms
	OpenKMS              bool                     // it's to open kms,default is false. https://help.aliyun.com/product/28933.html
	CacheDir             string                   // the directory for persist nacos service info,default value is current path
	DisableUseSnapShot   bool                     // It's a switch, default is false, means that when get remote config fail, use local cache file instead
	UpdateThreadNum      int                      // the number of goroutine for update nacos service info,default value is 20
	NotLoadCacheAtStart  bool                     // not to load persistent nacos service info in CacheDir at start time
	UpdateCacheWhenEmpty bool                     // update cache when get empty service instance from server
	Username             string                   // the username for nacos auth
	Password             string                   // the password for nacos auth
	LogDir               string                   // the directory for log, default is current path
	LogLevel             string                   // the level of log, it's must be debug,info,warn,error, default value is info
	ContextPath          string                   // the nacos server contextpath
	AppendToStdout       bool                     // if append log to stdout
	LogSampling          *ClientLogSamplingConfig // the sampling config of log
	LogRollingConfig     *ClientLogRollingConfig  // log rolling config
	TLSCfg               TLSConfig                // tls Config
	AsyncUpdateService   bool                     // open async update service by query
}

type ClientLogSamplingConfig struct {
	Initial    int           //the sampling initial of log
	Thereafter int           //the sampling thereafter of log
	Tick       time.Duration //the sampling tick of log
}

type ClientLogRollingConfig struct {
	// MaxSize is the maximum size in megabytes of the log file before it gets
	// rotated. It defaults to 100 megabytes.
	MaxSize int

	// MaxAge is the maximum number of days to retain old log files based on the
	// timestamp encoded in their filename.  Note that a day is defined as 24
	// hours and may not exactly correspond to calendar days due to daylight
	// savings, leap seconds, etc. The default is not to remove old log files
	// based on age.
	MaxAge int

	// MaxBackups is the maximum number of old log files to retain.  The default
	// is to retain all old log files (though MaxAge may still cause them to get
	// deleted.)
	MaxBackups int

	// LocalTime determines if the time used for formatting the timestamps in
	// backup files is the computer's local time.  The default is to use UTC
	// time.
	LocalTime bool

	// Compress determines if the rotated log files should be compressed
	// using gzip. The default is not to perform compression.
	Compress bool
}

type TLSConfig struct {
	Enable             bool   // enable tls
	CaFile             string // clients use when verifying server certificates
	CertFile           string // server use when verifying client certificates
	KeyFile            string // server use when verifying client certificates
	ServerNameOverride string // serverNameOverride is for testing only
}
