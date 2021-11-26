package naming_grpc

import (
	"reflect"
	"testing"

	"bou.ke/monkey"
	"github.com/nacos-group/nacos-sdk-go/v2/model"
	"github.com/nacos-group/nacos-sdk-go/v2/util"
	"github.com/stretchr/testify/assert"
)

func TestRedoSubscribeAfterCache(t *testing.T) {
	evListener := NewConnectionEventListener(&NamingGrpcProxy{})

	cases := []struct {
		serviceName string
		groupName   string
		clusters    string
	}{
		{"service-a", "group-a", ""},
		{"service-b", "group-b", "cluster-b"},
	}

	for _, v := range cases {
		monkey.PatchInstanceMethod(reflect.TypeOf(evListener.clientProxy),
			"Subscribe",
			func(_ *NamingGrpcProxy, serviceName, groupName, clusters string) (model.Service, error) {
				assert.Equal(t, serviceName, v.serviceName)
				assert.Equal(t, groupName, v.groupName)
				assert.Equal(t, clusters, v.clusters)
				return model.Service{}, nil
			})
		fullServiceName := util.GetGroupName(v.serviceName, v.groupName)
		evListener.CacheSubscriberForRedo(fullServiceName, v.clusters)
		evListener.redoSubscribe()
		evListener.RemoveSubscriberForRedo(fullServiceName, v.clusters)
		monkey.UnpatchAll()
	}
}
