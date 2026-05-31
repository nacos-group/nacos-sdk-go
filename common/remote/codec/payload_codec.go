package codec

import (
	"fmt"

	nacos_grpc_service "github.com/nacos-group/nacos-sdk-proto/go"
	"github.com/nacos-group/nacos-sdk-proto/go/ai"
	"github.com/nacos-group/nacos-sdk-proto/go/common"
	"github.com/nacos-group/nacos-sdk-proto/go/config"
	"github.com/nacos-group/nacos-sdk-proto/go/lock"
	"github.com/nacos-group/nacos-sdk-proto/go/naming"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
)

type PayloadCodec struct {
	registry map[string]func() proto.Message
}

func NewPayloadCodec() *PayloadCodec {
	c := &PayloadCodec{registry: make(map[string]func() proto.Message)}
	c.registerCommon()
	c.registerConfig()
	c.registerNaming()
	c.registerLock()
	c.registerAi()
	return c
}

func (c *PayloadCodec) Register(typeName string, factory func() proto.Message) {
	c.registry[typeName] = factory
}

func (c *PayloadCodec) Encode(typeName string, msg proto.Message, headers map[string]string, clientIp string) (*nacos_grpc_service.Payload, error) {
	jsonBytes, err := protojson.Marshal(msg)
	if err != nil {
		return nil, fmt.Errorf("encode %s: %w", typeName, err)
	}
	return &nacos_grpc_service.Payload{
		Metadata: &nacos_grpc_service.Metadata{
			Type:     typeName,
			ClientIp: clientIp,
			Headers:  headers,
		},
		Body: &anypb.Any{
			Value: jsonBytes,
		},
	}, nil
}

func (c *PayloadCodec) Decode(payload *nacos_grpc_service.Payload) (proto.Message, error) {
	typeName := payload.GetMetadata().GetType()
	factory, ok := c.registry[typeName]
	if !ok {
		return nil, fmt.Errorf("unknown message type: %s", typeName)
	}
	msg := factory()
	if err := protojson.Unmarshal(payload.GetBody().GetValue(), msg); err != nil {
		return nil, fmt.Errorf("decode %s: %w", typeName, err)
	}
	return msg, nil
}

func (c *PayloadCodec) registerCommon() {
	c.registry["ClientDetectionRequest"] = func() proto.Message { return &common.ClientDetectionRequest{} }
	c.registry["ClientDetectionResponse"] = func() proto.Message { return &common.ClientDetectionResponse{} }
	c.registry["ConnectResetRequest"] = func() proto.Message { return &common.ConnectResetRequest{} }
	c.registry["ConnectResetResponse"] = func() proto.Message { return &common.ConnectResetResponse{} }
	c.registry["ConnectionSetupRequest"] = func() proto.Message { return &common.ConnectionSetupRequest{} }
	c.registry["ErrorResponse"] = func() proto.Message { return &common.ErrorResponse{} }
	c.registry["HealthCheckRequest"] = func() proto.Message { return &common.HealthCheckRequest{} }
	c.registry["HealthCheckResponse"] = func() proto.Message { return &common.HealthCheckResponse{} }
	c.registry["PushAckRequest"] = func() proto.Message { return &common.PushAckRequest{} }
	c.registry["ServerCheckRequest"] = func() proto.Message { return &common.ServerCheckRequest{} }
	c.registry["ServerCheckResponse"] = func() proto.Message { return &common.ServerCheckResponse{} }
	c.registry["ServerLoaderInfoRequest"] = func() proto.Message { return &common.ServerLoaderInfoRequest{} }
	c.registry["ServerLoaderInfoResponse"] = func() proto.Message { return &common.ServerLoaderInfoResponse{} }
	c.registry["ServerReloadRequest"] = func() proto.Message { return &common.ServerReloadRequest{} }
	c.registry["ServerReloadResponse"] = func() proto.Message { return &common.ServerReloadResponse{} }
	c.registry["SetupAckRequest"] = func() proto.Message { return &common.SetupAckRequest{} }
	c.registry["SetupAckResponse"] = func() proto.Message { return &common.SetupAckResponse{} }
}

func (c *PayloadCodec) registerConfig() {
	c.registry["ConfigQueryRequest"] = func() proto.Message { return &config.ConfigQueryRequest{} }
	c.registry["ConfigQueryResponse"] = func() proto.Message { return &config.ConfigQueryResponse{} }
	c.registry["ConfigPublishRequest"] = func() proto.Message { return &config.ConfigPublishRequest{} }
	c.registry["ConfigPublishResponse"] = func() proto.Message { return &config.ConfigPublishResponse{} }
	c.registry["ConfigRemoveRequest"] = func() proto.Message { return &config.ConfigRemoveRequest{} }
	c.registry["ConfigRemoveResponse"] = func() proto.Message { return &config.ConfigRemoveResponse{} }
	c.registry["ConfigBatchListenRequest"] = func() proto.Message { return &config.ConfigBatchListenRequest{} }
	c.registry["ConfigChangeBatchListenResponse"] = func() proto.Message { return &config.ConfigChangeBatchListenResponse{} }
	c.registry["ConfigChangeNotifyRequest"] = func() proto.Message { return &config.ConfigChangeNotifyRequest{} }
	c.registry["ConfigChangeNotifyResponse"] = func() proto.Message { return &config.ConfigChangeNotifyResponse{} }
	c.registry["ConfigChangeClusterSyncRequest"] = func() proto.Message { return &config.ConfigChangeClusterSyncRequest{} }
	c.registry["ConfigChangeClusterSyncResponse"] = func() proto.Message { return &config.ConfigChangeClusterSyncResponse{} }
	c.registry["ClientConfigMetricRequest"] = func() proto.Message { return &config.ClientConfigMetricRequest{} }
	c.registry["ClientConfigMetricResponse"] = func() proto.Message { return &config.ClientConfigMetricResponse{} }
	c.registry["ConfigFuzzyWatchRequest"] = func() proto.Message { return &config.ConfigFuzzyWatchRequest{} }
	c.registry["ConfigFuzzyWatchResponse"] = func() proto.Message { return &config.ConfigFuzzyWatchResponse{} }
	c.registry["ConfigFuzzyWatchSyncRequest"] = func() proto.Message { return &config.ConfigFuzzyWatchSyncRequest{} }
	c.registry["ConfigFuzzyWatchSyncResponse"] = func() proto.Message { return &config.ConfigFuzzyWatchSyncResponse{} }
	c.registry["ConfigFuzzyWatchChangeNotifyRequest"] = func() proto.Message { return &config.ConfigFuzzyWatchChangeNotifyRequest{} }
	c.registry["ConfigFuzzyWatchChangeNotifyResponse"] = func() proto.Message { return &config.ConfigFuzzyWatchChangeNotifyResponse{} }
}

func (c *PayloadCodec) registerNaming() {
	c.registry["InstanceRequest"] = func() proto.Message { return &naming.InstanceRequest{} }
	c.registry["InstanceResponse"] = func() proto.Message { return &naming.InstanceResponse{} }
	c.registry["BatchInstanceRequest"] = func() proto.Message { return &naming.BatchInstanceRequest{} }
	c.registry["BatchInstanceResponse"] = func() proto.Message { return &naming.BatchInstanceResponse{} }
	c.registry["ServiceQueryRequest"] = func() proto.Message { return &naming.ServiceQueryRequest{} }
	c.registry["QueryServiceResponse"] = func() proto.Message { return &naming.QueryServiceResponse{} }
	c.registry["ServiceListRequest"] = func() proto.Message { return &naming.ServiceListRequest{} }
	c.registry["ServiceListResponse"] = func() proto.Message { return &naming.ServiceListResponse{} }
	c.registry["SubscribeServiceRequest"] = func() proto.Message { return &naming.SubscribeServiceRequest{} }
	c.registry["SubscribeServiceResponse"] = func() proto.Message { return &naming.SubscribeServiceResponse{} }
	c.registry["NotifySubscriberRequest"] = func() proto.Message { return &naming.NotifySubscriberRequest{} }
	c.registry["NotifySubscriberResponse"] = func() proto.Message { return &naming.NotifySubscriberResponse{} }
	c.registry["PersistentInstanceRequest"] = func() proto.Message { return &naming.PersistentInstanceRequest{} }
	c.registry["NamingFuzzyWatchRequest"] = func() proto.Message { return &naming.NamingFuzzyWatchRequest{} }
	c.registry["NamingFuzzyWatchResponse"] = func() proto.Message { return &naming.NamingFuzzyWatchResponse{} }
	c.registry["NamingFuzzyWatchSyncRequest"] = func() proto.Message { return &naming.NamingFuzzyWatchSyncRequest{} }
	c.registry["NamingFuzzyWatchSyncResponse"] = func() proto.Message { return &naming.NamingFuzzyWatchSyncResponse{} }
	c.registry["NamingFuzzyWatchChangeNotifyRequest"] = func() proto.Message { return &naming.NamingFuzzyWatchChangeNotifyRequest{} }
	c.registry["NamingFuzzyWatchChangeNotifyResponse"] = func() proto.Message { return &naming.NamingFuzzyWatchChangeNotifyResponse{} }
}

func (c *PayloadCodec) registerLock() {
	c.registry["LockOperationRequest"] = func() proto.Message { return &lock.LockOperationRequest{} }
	c.registry["LockOperationResponse"] = func() proto.Message { return &lock.LockOperationResponse{} }
}

func (c *PayloadCodec) registerAi() {
	c.registry["AgentEndpointRequest"] = func() proto.Message { return &ai.AgentEndpointRequest{} }
	c.registry["AgentEndpointResponse"] = func() proto.Message { return &ai.AgentEndpointResponse{} }
	c.registry["BatchAgentEndpointRequest"] = func() proto.Message { return &ai.BatchAgentEndpointRequest{} }
	c.registry["McpServerEndpointRequest"] = func() proto.Message { return &ai.McpServerEndpointRequest{} }
	c.registry["McpServerEndpointResponse"] = func() proto.Message { return &ai.McpServerEndpointResponse{} }
	c.registry["QueryAgentCardRequest"] = func() proto.Message { return &ai.QueryAgentCardRequest{} }
	c.registry["QueryAgentCardResponse"] = func() proto.Message { return &ai.QueryAgentCardResponse{} }
	c.registry["QueryMcpServerRequest"] = func() proto.Message { return &ai.QueryMcpServerRequest{} }
	c.registry["QueryMcpServerResponse"] = func() proto.Message { return &ai.QueryMcpServerResponse{} }
	c.registry["QueryPromptRequest"] = func() proto.Message { return &ai.QueryPromptRequest{} }
	c.registry["QueryPromptResponse"] = func() proto.Message { return &ai.QueryPromptResponse{} }
	c.registry["ReleaseAgentCardRequest"] = func() proto.Message { return &ai.ReleaseAgentCardRequest{} }
	c.registry["ReleaseAgentCardResponse"] = func() proto.Message { return &ai.ReleaseAgentCardResponse{} }
	c.registry["ReleaseMcpServerRequest"] = func() proto.Message { return &ai.ReleaseMcpServerRequest{} }
	c.registry["ReleaseMcpServerResponse"] = func() proto.Message { return &ai.ReleaseMcpServerResponse{} }
}
