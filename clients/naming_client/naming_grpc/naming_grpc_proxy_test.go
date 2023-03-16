package naming_grpc

import "github.com/nacos-group/nacos-sdk-go/v2/model"

type MockNamingGrpc struct {
}

func (m *MockNamingGrpc) RegisterInstance(serviceName string, groupName string, instance model.Instance) (bool, error) {
	return true, nil
}

func (m *MockNamingGrpc) BatchRegisterInstance(serviceName string, groupName string, instances []model.Instance) (bool, error) {
	return true, nil
}

func (m *MockNamingGrpc) DeregisterInstance(serviceName string, groupName string, instance model.Instance) (bool, error) {
	return true, nil
}

func (m *MockNamingGrpc) GetServiceList(pageNo uint32, pageSize uint32, groupName string, selector *model.ExpressionSelector) (model.ServiceList, error) {
	return model.ServiceList{Doms: []string{""}}, nil
}

func (m *MockNamingGrpc) ServerHealthy() bool {
	return true
}

func (m *MockNamingGrpc) QueryInstancesOfService(serviceName, groupName, clusters string, udpPort int, healthyOnly bool) (*model.Service, error) {
	return &model.Service{}, nil
}

func (m *MockNamingGrpc) Subscribe(serviceName, groupName, clusters string) (model.Service, error) {
	return model.Service{}, nil
}

func (m *MockNamingGrpc) Unsubscribe(serviceName, groupName, clusters string) error {
	return nil
}

func (m *MockNamingGrpc) CloseClient() {}
