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

const (
	KEY_USERNAME                = "username"
	KEY_PASSWORD                = "password"
	KEY_ENDPOINT                = "endpoint"
	KEY_NAME_SPACE              = "namespace"
	KEY_ACCESS_KEY              = "accessKey"
	KEY_SECRET_KEY              = "secretKey"
	KEY_SERVER_ADDR             = "serverAddr"
	KEY_CONTEXT_PATH            = "contextPath"
	KEY_ENCODE                  = "encode"
	KEY_DATA_ID                 = "dataId"
	KEY_GROUP                   = "group"
	KEY_TENANT                  = "tenant"
	KEY_DESC                    = "desc"
	KEY_APP_NAME                = "appName"
	KEY_CONTENT                 = "content"
	KEY_TIMEOUT_MS              = "timeoutMs"
	KEY_LISTEN_INTERVAL         = "listenInterval"
	KEY_SERVER_CONFIGS          = "serverConfigs"
	KEY_CLIENT_CONFIG           = "clientConfig"
	KEY_TOKEN                   = "token"
	KEY_ACCESS_TOKEN            = "accessToken"
	KEY_TOKEN_TTL               = "tokenTtl"
	KEY_GLOBAL_ADMIN            = "globalAdmin"
	KEY_TOKEN_REFRESH_WINDOW    = "tokenRefreshWindow"
	WEB_CONTEXT                 = "/nacos"
	CONFIG_BASE_PATH            = "/v1/cs"
	CONFIG_PATH                 = CONFIG_BASE_PATH + "/configs"
	CONFIG_AGG_PATH             = "/datum.do"
	CONFIG_LISTEN_PATH          = CONFIG_BASE_PATH + "/configs/listener"
	SERVICE_BASE_PATH           = "/v1/ns"
	SERVICE_PATH                = SERVICE_BASE_PATH + "/instance"
	SERVICE_INFO_PATH           = SERVICE_BASE_PATH + "/service"
	SERVICE_SUBSCRIBE_PATH      = SERVICE_PATH + "/list"
	NAMESPACE_PATH              = "/v1/console/namespaces"
	SPLIT_CONFIG                = string(rune(1))
	SPLIT_CONFIG_INNER          = string(rune(2))
	KEY_LISTEN_CONFIGS          = "Listening-Configs"
	KEY_SERVICE_NAME            = "serviceName"
	KEY_IP                      = "ip"
	KEY_PORT                    = "port"
	KEY_WEIGHT                  = "weight"
	KEY_ENABLE                  = "enable"
	KEY_HEALTHY                 = "healthy"
	KEY_METADATA                = "metadata"
	KEY_CLUSTER_NAME            = "clusterName"
	KEY_CLUSTER                 = "cluster"
	KEY_BEAT                    = "beat"
	KEY_DOM                     = "dom"
	DEFAULT_CONTEXT_PATH        = "/nacos"
	CLIENT_VERSION              = "Nacos-Go-Client:v2.2.2"
	REQUEST_DOMAIN_RETRY_TIME   = 3
	SERVICE_INFO_SPLITER        = "@@"
	CONFIG_INFO_SPLITER         = "@@"
	DEFAULT_NAMESPACE_ID        = "public"
	DEFAULT_GROUP               = "DEFAULT_GROUP"
	NAMING_INSTANCE_ID_SPLITTER = "#"
	DefaultClientErrorCode      = "SDK.NacosError"
	DEFAULT_SERVER_SCHEME       = "http"
	HTTPS_SERVER_SCHEME         = "https"
	LABEL_SOURCE                = "source"
	LABEL_SOURCE_SDK            = "sdk"
	LABEL_MODULE                = "module"
	LABEL_MODULE_CONFIG         = "config"
	LABEL_MODULE_NAMING         = "naming"
	RESPONSE_CODE_SUCCESS       = 200
	UN_REGISTER                 = 301
	KEEP_ALIVE_TIME             = 5
	DEFAULT_TIMEOUT_MILLS       = 3000
	ALL_SYNC_INTERNAL           = 5 * time.Minute
	CLIENT_APPNAME_HEADER       = "Client-AppName"
	APPNAME_HEADER              = "AppName"
	CLIENT_REQUEST_TS_HEADER    = "Client-RequestTS"
	CLIENT_REQUEST_TOKEN_HEADER = "Client-RequestToken"
	EX_CONFIG_INFO              = "exConfigInfo"
	CHARSET_KEY                 = "charset"
	LOG_FILE_NAME               = "nacos-sdk.log"
	HTTPS_SERVER_PORT           = 443
	GRPC                        = "grpc"
	FAILOVER_FILE_SUFFIX        = "_failover"
	RpcPortOffset               = 1000
)
