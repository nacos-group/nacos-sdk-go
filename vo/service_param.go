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

import "github.com/nacos-group/nacos-sdk-go/v2/model"

type RegisterInstanceParam struct {
	Ip          string            `param:"ip"`          //required
	Port        uint64            `param:"port"`        //required
	Weight      float64           `param:"weight"`      //required,it must be lager than 0
	Enable      bool              `param:"enabled"`     //required,the instance can be access or not
	Healthy     bool              `param:"healthy"`     //required,the instance is health or not
	Metadata    map[string]string `param:"metadata"`    //optional
	ClusterName string            `param:"clusterName"` //optional
	ServiceName string            `param:"serviceName"` //required
	GroupName   string            `param:"groupName"`   //optional,default:DEFAULT_GROUP
	Ephemeral   bool              `param:"ephemeral"`   //optional
}

type BatchRegisterInstanceParam struct {
	ServiceName string                  `param:"serviceName"` //required
	GroupName   string                  `param:"groupName"`   //optional,default:DEFAULT_GROUP
	Instances   []RegisterInstanceParam //required
}

type DeregisterInstanceParam struct {
	Ip          string `param:"ip"`          //required
	Port        uint64 `param:"port"`        //required
	Cluster     string `param:"cluster"`     //optional
	ServiceName string `param:"serviceName"` //required
	GroupName   string `param:"groupName"`   //optional,default:DEFAULT_GROUP
	Ephemeral   bool   `param:"ephemeral"`   //optional
}

type UpdateInstanceParam struct {
	Ip          string            `param:"ip"`          //required
	Port        uint64            `param:"port"`        //required
	Weight      float64           `param:"weight"`      //required,it must be lager than 0
	Enable      bool              `param:"enabled"`     //required,the instance can be access or not
	Healthy     bool              `param:"healthy"`     //required,the instance is health or not
	Metadata    map[string]string `param:"metadata"`    //optional
	ClusterName string            `param:"clusterName"` //optional
	ServiceName string            `param:"serviceName"` //required
	GroupName   string            `param:"groupName"`   //optional,default:DEFAULT_GROUP
	Ephemeral   bool              `param:"ephemeral"`   //optional
}

type GetServiceParam struct {
	Clusters    []string `param:"clusters"`    //optional
	ServiceName string   `param:"serviceName"` //required
	GroupName   string   `param:"groupName"`   //optional,default:DEFAULT_GROUP
}

type GetAllServiceInfoParam struct {
	NameSpace string `param:"nameSpace"` //optional, namespaceId default:public
	GroupName string `param:"groupName"` //optional,default:DEFAULT_GROUP
	PageNo    uint32 `param:"pageNo"`    //optional,default:1
	PageSize  uint32 `param:"pageSize"`  //optional,default:10
}

type SubscribeParam struct {
	ServiceName       string                                     `param:"serviceName"` //required
	Clusters          []string                                   `param:"clusters"`    //optional
	GroupName         string                                     `param:"groupName"`   //optional,default:DEFAULT_GROUP
	SubscribeCallback func(services []model.Instance, err error) //required
}

type SelectAllInstancesParam struct {
	Clusters    []string `param:"clusters"`    //optional
	ServiceName string   `param:"serviceName"` //required
	GroupName   string   `param:"groupName"`   //optional,default:DEFAULT_GROUP
}

type SelectInstancesParam struct {
	Clusters    []string `param:"clusters"`    //optional
	ServiceName string   `param:"serviceName"` //required
	GroupName   string   `param:"groupName"`   //optional,default:DEFAULT_GROUP
	HealthyOnly bool     `param:"healthyOnly"` //optional,value = true return only healthy instance, value = false return only unHealthy instance
}

type SelectOneHealthInstanceParam struct {
	Clusters    []string `param:"clusters"`    //optional
	ServiceName string   `param:"serviceName"` //required
	GroupName   string   `param:"groupName"`   //optional,default:DEFAULT_GROUP
}
