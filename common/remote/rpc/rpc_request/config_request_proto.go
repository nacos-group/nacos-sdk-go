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
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package rpc_request

import (
	"github.com/nacos-group/nacos-sdk-proto/go/config"
	"google.golang.org/protobuf/proto"
)

func (r *ConfigQueryRequest) ProtoMessage() proto.Message {
	return &config.ConfigQueryRequest{
		DataId: r.DataId,
		Group:  r.Group,
		Tenant: r.Tenant,
		Tag:    r.Tag,
	}
}

func (r *ConfigPublishRequest) ProtoMessage() proto.Message {
	return &config.ConfigPublishRequest{
		DataId:      r.DataId,
		Group:       r.Group,
		Tenant:      r.Tenant,
		Content:     r.Content,
		CasMd5:      r.CasMd5,
		AdditionMap: r.AdditionMap,
	}
}

func (r *ConfigRemoveRequest) ProtoMessage() proto.Message {
	return &config.ConfigRemoveRequest{
		DataId: r.DataId,
		Group:  r.Group,
		Tenant: r.Tenant,
	}
}

func (r *ConfigBatchListenRequest) ProtoMessage() proto.Message {
	ctxs := make([]*config.ConfigBatchListenRequestConfigListenContext, 0, len(r.ConfigListenContexts))
	for _, c := range r.ConfigListenContexts {
		ctxs = append(ctxs, &config.ConfigBatchListenRequestConfigListenContext{
			Group:  c.Group,
			Md5:    c.Md5,
			DataId: c.DataId,
			Tenant: c.Tenant,
		})
	}
	return &config.ConfigBatchListenRequest{
		Listen:               r.Listen,
		ConfigListenContexts: ctxs,
	}
}
