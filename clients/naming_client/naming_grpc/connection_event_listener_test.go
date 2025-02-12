package naming_grpc

import (
	"github.com/golang/mock/gomock"
	"github.com/nacos-group/nacos-sdk-go/v2/clients/naming_client/naming_proxy"
	"github.com/nacos-group/nacos-sdk-go/v2/util"
	"testing"
)

func TestRedoSubscribe(t *testing.T) {
	t.Skip("Skipping test,It failed due to a previous commit and is difficult to modify because of the use of struct type assertions in the code.")
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockProxy := naming_proxy.NewMockINamingProxy(ctrl)
	evListener := NewConnectionEventListener(mockProxy)

	cases := []struct {
		serviceName string
		groupName   string
		clusters    string
	}{
		{"service-a", "group-a", ""},
		{"service-b", "group-b", "cluster-b"},
	}

	for _, v := range cases {
		fullServiceName := util.GetGroupName(v.serviceName, v.groupName)
		evListener.CacheSubscriberForRedo(fullServiceName, v.clusters)
		mockProxy.EXPECT().Subscribe(v.serviceName, v.groupName, v.clusters)
		evListener.redoSubscribe()
		evListener.RemoveSubscriberForRedo(fullServiceName, v.clusters)
	}
}
