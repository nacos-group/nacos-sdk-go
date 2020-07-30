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

package vo

import "github.com/nacos-group/nacos-sdk-go/model"

type RegisterInstanceParam struct {
	Ip          string            `param:"ip"`          //required
	Port        uint64            `param:"port"`        //required
	Tenant      string            `param:"tenant"`      //optional
	Weight      float64           `param:"weight"`      //required,it must be lager than 0
	Enable      bool              `param:"enabled"`     //required,the instance can be access or not
	Healthy     bool              `param:"healthy"`     //required,the instance is health or not
	Metadata    map[string]string `param:"metadata"`    //optional
	ClusterName string            `param:"clusterName"` //optional,default:DEFAULT
	ServiceName string            `param:"serviceName"` //required
	GroupName   string            `param:"groupName"`   //optional,default:DEFAULT_GROUP
	Ephemeral   bool              `param:"ephemeral"`   //optional
}

type DeregisterInstanceParam struct {
	Ip          string `param:"ip"`          //required
	Port        uint64 `param:"port"`        //required
	Tenant      string `param:"tenant"`      //optional
	Cluster     string `param:"cluster"`     //optional,default:DEFAULT
	ServiceName string `param:"serviceName"` //required
	GroupName   string `param:"groupName"`   //optional,default:DEFAULT_GROUP
	Ephemeral   bool   `param:"ephemeral"`   //optional
}

type GetServiceParam struct {
	Clusters    []string `param:"clusters"`    //optional,default:DEFAULT
	ServiceName string   `param:"serviceName"` //required
	GroupName   string   `param:"groupName"`   //optional,default:DEFAULT_GROUP
}

type GetAllServiceInfoParam struct {
	NameSpace string `param:"nameSpace"`
	GroupName string `param:"groupName"`
	PageNo    uint32 `param:"pageNo"`
	PageSize  uint32 `param:"pageSize"`
}

type GetServiceListParam struct {
	StartPage   uint32 `param:"startPg"`
	PageSize    uint32 `param:"pgSize"`
	Keyword     string `param:"keyword"`
	NamespaceId string `param:"namespaceId"`
}

type GetServiceInstanceParam struct {
	Tenant      string `param:"tenant"`
	HealthyOnly bool   `param:"healthyOnly"`
	Cluster     string `param:"cluster"`
	ServiceName string `param:"serviceName"`
	Ip          string `param:"ip"`
	Port        uint64 `param:"port"`
}

type GetServiceDetailParam struct {
	ServiceName string `param:"serviceName"`
}

type SubscribeParam struct {
	ServiceName       string                                             `param:"serviceName"` //required
	Clusters          []string                                           `param:"clusters"`    //optional,default:DEFAULT
	GroupName         string                                             `param:"groupName"`   //optional,default:DEFAULT_GROUP
	SubscribeCallback func(services []model.SubscribeService, err error) //required
}

type SelectAllInstancesParam struct {
	Clusters    []string `param:"clusters"`    //optional,default:DEFAULT
	ServiceName string   `param:"serviceName"` //required
	GroupName   string   `param:"groupName"`   //optional,default:DEFAULT_GROUP
}

type SelectInstancesParam struct {
	Clusters    []string `param:"clusters"`
	ServiceName string   `param:"serviceName"`
	GroupName   string   `param:"groupName"`
	HealthyOnly bool     `param:"healthyOnly"`
}

type SelectOneHealthInstanceParam struct {
	Clusters    []string `param:"clusters"`
	ServiceName string   `param:"serviceName"`
	GroupName   string   `param:"groupName"`
}
