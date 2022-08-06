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

import (
	"github.com/nacos-group/nacos-sdk-go/common/logger"

	"gopkg.in/natefinch/lumberjack.v2"
)

type ServerConfig struct {
	Scheme      string //the nacos server scheme
	ContextPath string //the nacos server contextpath
	IpAddr      string //the nacos server address
	Port        uint64 //the nacos server port
}

type ClientConfig struct {
	TimeoutMs            uint64                 // timeout for requesting Nacos server, default value is 10000ms
	ListenInterval       uint64                 // Deprecated
	BeatInterval         int64                  // the time interval for sending beat to server,default value is 5000ms
	NamespaceId          string                 // the namespaceId of Nacos.When namespace is public, fill in the blank string here.
	AppName              string                 // the appName
	Endpoint             string                 // the endpoint for get Nacos server addresses
	RegionId             string                 // the regionId for kms
	AccessKey            string                 // the AccessKey for kms
	SecretKey            string                 // the SecretKey for kms
	OpenKMS              bool                   // it's to open kms,default is false. https://help.aliyun.com/product/28933.html
	CacheDir             string                 // the directory for persist nacos service info,default value is current path
	UpdateThreadNum      int                    // the number of gorutine for update nacos service info,default value is 20
	NotLoadCacheAtStart  bool                   // not to load persistent nacos service info in CacheDir at start time
	UpdateCacheWhenEmpty bool                   // update cache when get empty service instance from server
	Username             string                 // the username for nacos auth
	Password             string                 // the password for nacos auth
	LogDir               string                 // the directory for log, default is current path
	LogLevel             string                 // the level of log, it's must be debug,info,warn,error, default value is info
	LogSampling          *logger.SamplingConfig // the sampling config of log
	ContextPath          string                 // the nacos server contextpath
	LogRollingConfig     *lumberjack.Logger     // the log rolling config
	CustomLogger         logger.Logger          // the custom log interface ,With a custom Logger (nacos sdk will not provide log cutting and archiving capabilities)
	AppendToStdout       bool                   // append log to stdout
}
