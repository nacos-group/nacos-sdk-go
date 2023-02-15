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

package naming_proxy

import (
	"github.com/nacos-group/nacos-sdk-go/v2/model"
)

// INamingProxy ...
type INamingProxy interface {
	RegisterInstance(serviceName string, groupName string, instance model.Instance) (bool, error)

	BatchRegisterInstance(serviceName string, groupName string, instances []model.Instance) (bool, error)

	DeregisterInstance(serviceName string, groupName string, instance model.Instance) (bool, error)

	GetServiceList(pageNo uint32, pageSize uint32, groupName, namespaceId string, selector *model.ExpressionSelector) (model.ServiceList, error)

	ServerHealthy() bool

	QueryInstancesOfService(serviceName, groupName, clusters string, udpPort int, healthyOnly bool) (*model.Service, error)

	Subscribe(serviceName, groupName, clusters string) (model.Service, error)

	Unsubscribe(serviceName, groupName, clusters string) error

	CloseClient()
}
