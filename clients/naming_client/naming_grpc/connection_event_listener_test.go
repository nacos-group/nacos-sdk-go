package naming_grpc

import (
	"testing"
)

func TestRedoSubscribe(t *testing.T) {
	//ctrl := gomock.NewController(t)
	//defer ctrl.Finish()
	//
	//mockProxy := naming_proxy.NewMockINamingProxy(ctrl)
	//evListener := NewConnectionEventListener(mockProxy)
	//
	//cases := []struct {
	//	serviceName string
	//	groupName   string
	//	clusters    string
	//}{
	//	{"service-a", "group-a", ""},
	//	{"service-b", "group-b", "cluster-b"},
	//}
	//
	//for _, v := range cases {
	//	fullServiceName := util.GetGroupName(v.serviceName, v.groupName)
	//	evListener.CacheSubscriberForRedo(fullServiceName, v.clusters)
	//	mockProxy.EXPECT().Subscribe(v.serviceName, v.groupName, v.clusters)
	//	evListener.redoSubscribe()
	//	evListener.RemoveSubscriberForRedo(fullServiceName, v.clusters)
	//}
}
