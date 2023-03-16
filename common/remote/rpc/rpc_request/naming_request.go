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

package rpc_request

import (
	"fmt"
	"strconv"
	"time"

	"github.com/nacos-group/nacos-sdk-go/v2/model"
)

type NamingRequest struct {
	*Request
	Namespace   string `json:"namespace"`
	ServiceName string `json:"serviceName"`
	GroupName   string `json:"groupName"`
	Module      string `json:"module"`
}

func NewNamingRequest(namespace, serviceName, groupName string) *NamingRequest {
	request := Request{
		Headers: make(map[string]string, 8),
	}
	return &NamingRequest{
		Request:     &request,
		Namespace:   namespace,
		ServiceName: serviceName,
		GroupName:   groupName,
		Module:      "naming",
	}
}

func (r *NamingRequest) GetStringToSign() string {
	data := strconv.FormatInt(time.Now().Unix()*1000, 10)
	if r.ServiceName != "" || r.GroupName != "" {
		data = fmt.Sprintf("%s@@%s@@%s", data, r.GroupName, r.ServiceName)
	}
	return data
}

type InstanceRequest struct {
	*NamingRequest
	Type     string         `json:"type"`
	Instance model.Instance `json:"instance"`
}

func NewInstanceRequest(namespace, serviceName, groupName, Type string, instance model.Instance) *InstanceRequest {
	return &InstanceRequest{
		NamingRequest: NewNamingRequest(namespace, serviceName, groupName),
		Type:          Type,
		Instance:      instance,
	}
}

func (r *InstanceRequest) GetRequestType() string {
	return "InstanceRequest"
}

type BatchInstanceRequest struct {
	*NamingRequest
	Type      string           `json:"type"`
	Instances []model.Instance `json:"instances"`
}

func NewBatchInstanceRequest(namespace, serviceName, groupName, Type string, instances []model.Instance) *BatchInstanceRequest {
	return &BatchInstanceRequest{
		NamingRequest: NewNamingRequest(namespace, serviceName, groupName),
		Type:          Type,
		Instances:     instances,
	}
}

func (r *BatchInstanceRequest) GetRequestType() string {
	return "BatchInstanceRequest"
}

type NotifySubscriberRequest struct {
	*NamingRequest
	ServiceInfo model.Service `json:"serviceInfo"`
}

func (r *NotifySubscriberRequest) GetRequestType() string {
	return "NotifySubscriberRequest"
}

type ServiceListRequest struct {
	*NamingRequest
	PageNo   int    `json:"pageNo"`
	PageSize int    `json:"pageSize"`
	Selector string `json:"selector"`
}

func NewServiceListRequest(namespace, serviceName, groupName string, pageNo, pageSize int, selector string) *ServiceListRequest {
	return &ServiceListRequest{
		NamingRequest: NewNamingRequest(namespace, serviceName, groupName),
		PageNo:        pageNo,
		PageSize:      pageSize,
		Selector:      selector,
	}
}

func (r *ServiceListRequest) GetRequestType() string {
	return "ServiceListRequest"
}

type SubscribeServiceRequest struct {
	*NamingRequest
	Subscribe bool   `json:"subscribe"`
	Clusters  string `json:"clusters"`
}

func NewSubscribeServiceRequest(namespace, serviceName, groupName, clusters string, subscribe bool) *SubscribeServiceRequest {
	return &SubscribeServiceRequest{
		NamingRequest: NewNamingRequest(namespace, serviceName, groupName),
		Subscribe:     subscribe,
		Clusters:      clusters,
	}
}

func (r *SubscribeServiceRequest) GetRequestType() string {
	return "SubscribeServiceRequest"
}

type ServiceQueryRequest struct {
	*NamingRequest
	Cluster     string `json:"cluster"`
	HealthyOnly bool   `json:"healthyOnly"`
	UdpPort     int    `json:"udpPort"`
}

func NewServiceQueryRequest(namespace, serviceName, groupName, cluster string, healthyOnly bool, udpPort int) *ServiceQueryRequest {
	return &ServiceQueryRequest{
		NamingRequest: NewNamingRequest(namespace, serviceName, groupName),
		Cluster:       cluster,
		HealthyOnly:   healthyOnly,
		UdpPort:       udpPort,
	}
}

func (r *ServiceQueryRequest) GetRequestType() string {
	return "ServiceQueryRequest"
}
