package naming_client

import (
	"encoding/json"
	"testing"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/nacos-group/nacos-sdk-go/v2/model"
	"github.com/nacos-group/nacos-sdk-go/v2/vo"
	"github.com/pkg/errors"
	"github.com/smartystreets/goconvey/convey"
)

var testRawInstances = []model.Instance{
	{
		InstanceId: "1",
		Ip:         "127.0.0.1",
		Port:       8080,
		Weight:     1,
		Enable:     true,
		Healthy:    true,
		Metadata: map[string]string{
			"version": "1.0.0",
		},
		ClusterName: "dev01",
		ServiceName: "webserver",
		Ephemeral:   true,
	},
	{
		InstanceId: "2",
		Ip:         "127.0.0.2",
		Port:       8080,
		Weight:     1,
		Enable:     true,
		Healthy:    false,
		Metadata: map[string]string{
			"version": "1.0.0",
		},
		ClusterName: "dev01",
		ServiceName: "webserver",
		Ephemeral:   true,
	},
	{
		InstanceId: "3",
		Ip:         "127.0.0.3",
		Port:       8080,
		Weight:     1,
		Enable:     false,
		Healthy:    true,
		Metadata: map[string]string{
			"version": "1.0.0",
		},
		ClusterName: "dev01",
		ServiceName: "webserver",
		Ephemeral:   true,
	},
	{
		InstanceId: "4",
		Ip:         "127.0.0.4",
		Port:       8080,
		Weight:     1,
		Enable:     true,
		Healthy:    true,
		Metadata: map[string]string{
			"version": "1.0.1",
		},
		ClusterName: "dev01",
		ServiceName: "webserver",
		Ephemeral:   true,
	},
	{
		InstanceId: "5",
		Ip:         "127.0.0.5",
		Port:       8080,
		Weight:     1,
		Enable:     true,
		Healthy:    true,
		Metadata: map[string]string{
			"version": "1.0.2",
		},
		ClusterName: "dev01",
		ServiceName: "webserver",
		Ephemeral:   true,
	},
	{
		InstanceId: "6",
		Ip:         "127.0.0.6",
		Port:       8080,
		Weight:     1,
		Enable:     true,
		Healthy:    true,
		Metadata: map[string]string{
			"version": "1.0.1",
		},
		ClusterName: "dev01",
		ServiceName: "webserver",
		Ephemeral:   true,
	},
	{
		InstanceId: "7",
		Ip:         "127.0.0.7",
		Port:       8080,
		Weight:     1,
		Enable:     true,
		Healthy:    true,
		Metadata: map[string]string{
			"version": "1.0.0",
		},
		ClusterName: "dev01",
		ServiceName: "webserver",
		Ephemeral:   true,
	},
	{
		InstanceId: "8",
		Ip:         "127.0.0.8",
		Port:       8080,
		Weight:     1,
		Enable:     true,
		Healthy:    false,
		Metadata: map[string]string{
			"version": "1.0.3",
		},
		ClusterName: "dev01",
		ServiceName: "webserver",
		Ephemeral:   true,
	},
	{
		InstanceId: "9",
		Ip:         "127.0.0.9",
		Port:       8080,
		Weight:     1,
		Enable:     true,
		Healthy:    false,
		Metadata: map[string]string{
			"version": "1.0.0",
		},
		ClusterName: "dev02",
		ServiceName: "webserver",
		Ephemeral:   true,
	},
}

func mockSelectInstances(param vo.SelectInstancesParam) ([]model.Instance, error) {
	result := []model.Instance{}
	if testRawInstances == nil || len(testRawInstances) == 0 {
		return result, errors.New("instance list is empty!")
	}
	for _, host := range testRawInstances {
		isInCluster := false
		for _, c := range param.Clusters {
			if host.ClusterName == c {
				isInCluster = true
				continue
			}
		}
		if !isInCluster {
			continue
		}
		if host.Healthy == param.HealthyOnly && host.Enable && host.Weight > 0 {
			result = append(result, host)
		}
	}
	return result, nil
}

func TestDiscover(t *testing.T) {
	convey.Convey("TestDiscover", t, func() {
		c := &NamingClient{}
		invokeFunc := gomonkey.ApplyMethodFunc(c, "SelectInstances", mockSelectInstances)
		defer invokeFunc.Reset()
		convey.Convey("TestDiscover", func() {
			res, err := c.Discover(
				WithGroup("a"),
				WithService("webserver"),
				WithCluster("dev01"))
			convey.So(err, convey.ShouldBeNil)
			convey.So(len(res), convey.ShouldEqual, 5)
		})

		convey.Convey("TestDiscoverOne", func() {
			for i := 0; i < 10; i++ {
				res, err := c.DiscoverOne(
					WithGroup("a"),
					WithService("webserver"),
					WithCluster("dev01"))
				convey.So(err, convey.ShouldBeNil)
				bs, _ := json.Marshal(res)
				println(string(bs))
			}
		})
	})
}
