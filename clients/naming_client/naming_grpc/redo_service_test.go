package naming_grpc

import (
	"context"
	"github.com/nacos-group/nacos-sdk-go/v2/model"
	"github.com/nacos-group/nacos-sdk-go/v2/util"
	"testing"

	. "github.com/agiledragon/gomonkey/v2"
	. "github.com/smartystreets/goconvey/convey"
)

func TestRedoSubscribe(t *testing.T) {
	Convey("to subscriber", t, func() {
		ctx, cancelFunc := context.WithCancel(context.Background())
		defer cancelFunc()
		subCalled := false
		var mock *NamingGrpcProxy
		evListener := NewRedoService(ctx, mock)
		patch := ApplyMethod(mock, "DoSubscribe", func(_ *NamingGrpcProxy, serviceName, groupName, clusters string) (model.Service, error) {
			subCalled = true
			evListener.SubscribeRegistered(serviceName, groupName, clusters)
			return model.Service{}, nil
		})
		defer patch.Reset()
		unSubCalled := false
		patch = ApplyMethod(mock, "DoUnSubscribe", func(_ *NamingGrpcProxy, serviceName, groupName, clusters string) error {
			unSubCalled = true
			evListener.SubscribeDeRegistered(serviceName, groupName, clusters)
			return nil
		})

		subscribeCases := []struct {
			serviceName string
			groupName   string
			clusters    string
		}{
			{"service-a", "group-a", ""},
			{"service-b", "group-b", "cluster-b"},
		}

		for _, v := range subscribeCases {
			evListener.CacheSubscriberForRedo(v.serviceName, v.groupName, v.clusters)
			evListener.OnConnected()
			evListener.task.DoRedo()

			So(subCalled, ShouldBeTrue)
			subCalled = false
			evListener.SubscribeDeRegister(v.serviceName, v.groupName, v.clusters)
			evListener.task.DoRedo()
			So(unSubCalled, ShouldBeTrue)

			evListener.task.DoRedo()
			So(evListener.IsSubscriberCached(v.serviceName, v.groupName, v.clusters), ShouldBeFalse)
			evListener.RemoveSubscriberForRedo(v.serviceName, v.groupName, v.clusters)
		}
	})

}

func TestRedoInstance(t *testing.T) {
	Convey("to instance", t, func() {
		ctx, cancelFunc := context.WithCancel(context.Background())
		defer cancelFunc()
		registerCalled := false
		var mock *NamingGrpcProxy
		evListener := NewRedoService(ctx, mock)
		patch := ApplyMethod(mock, "DoRegisterInstance", func(_ *NamingGrpcProxy, serviceName, groupName string, instance model.Instance) error {
			registerCalled = true
			evListener.InstanceRegistered(serviceName, groupName)
			return nil
		})
		defer patch.Reset()
		deregisterCalled := false
		patch = ApplyMethod(mock, "DoDeRegisterInstance", func(_ *NamingGrpcProxy, serviceName, groupName string, instance model.Instance) error {
			deregisterCalled = true
			evListener.InstanceDeRegistered(serviceName, groupName)
			return nil
		})

		instanceCases := []struct {
			serviceName string
			groupName   string
			ins         model.Instance
		}{
			{"service-a", "group-a", model.Instance{}},
			{"service-b", "group-b", model.Instance{}},
		}

		for _, v := range instanceCases {
			evListener.CacheInstanceForRedo(v.serviceName, v.groupName, v.ins)
			evListener.OnConnected()
			evListener.task.DoRedo()

			So(registerCalled, ShouldBeTrue)
			registerCalled = false
			evListener.InstanceDeRegister(v.serviceName, v.groupName)
			evListener.task.DoRedo()
			So(deregisterCalled, ShouldBeTrue)

			evListener.task.DoRedo()
			_, ok := evListener.registeredRedoInstanceCached.Load(util.GetGroupName(v.serviceName, v.groupName))
			So(ok, ShouldBeFalse)
			evListener.RemoveInstanceForRedo(v.serviceName, v.groupName)
		}
	})
}

func TestRedoInstances(t *testing.T) {
	Convey("to instance", t, func() {
		ctx, cancelFunc := context.WithCancel(context.Background())
		defer cancelFunc()
		registerCalled := false
		var mock *NamingGrpcProxy
		evListener := NewRedoService(ctx, mock)
		patch := ApplyMethod(mock, "DoBatchRegisterInstance", func(_ *NamingGrpcProxy, serviceName, groupName string, instances []model.Instance) error {
			registerCalled = true
			evListener.InstanceRegistered(serviceName, groupName)
			return nil
		})
		defer patch.Reset()
		deregisterCalled := false
		patch = ApplyMethod(mock, "DoDeRegisterInstance", func(_ *NamingGrpcProxy, serviceName, groupName string, instance model.Instance) error {
			deregisterCalled = true
			evListener.InstanceDeRegistered(serviceName, groupName)
			return nil
		})

		instanceCases := []struct {
			serviceName string
			groupName   string
			ins         []model.Instance
		}{
			{"service-a", "group-a", []model.Instance{}},
			{"service-b", "group-b", []model.Instance{}},
		}

		for _, v := range instanceCases {
			evListener.CacheInstancesForRedo(v.serviceName, v.groupName, v.ins)
			evListener.OnConnected()
			evListener.task.DoRedo()

			So(registerCalled, ShouldBeTrue)
			registerCalled = false
			evListener.InstanceDeRegister(v.serviceName, v.groupName)
			evListener.task.DoRedo()
			So(deregisterCalled, ShouldBeTrue)

			evListener.task.DoRedo()
			_, ok := evListener.registeredRedoInstanceCached.Load(util.GetGroupName(v.serviceName, v.groupName))
			So(ok, ShouldBeFalse)
			evListener.RemoveInstanceForRedo(v.serviceName, v.groupName)
		}
	})
}
