package naming_grpc

import (
	"github.com/golang/mock/gomock"
	"github.com/nacos-group/nacos-sdk-go/v2/clients/naming_client/naming_proxy"
	"github.com/nacos-group/nacos-sdk-go/v2/util"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestSubscriberRedoRegisteredState(t *testing.T) {
	evListener := NewConnectionEventListener(nil)
	fullServiceName := util.GetGroupName("service-a", "group-a")
	key := util.GetServiceCacheKey(fullServiceName, "")

	evListener.CacheSubscriberForRedo(fullServiceName, "")
	assert.True(t, evListener.IsSubscriberCached(key))
	assert.False(t, evListener.IsSubscriberRegistered(key), "a cached redo intent must not be treated as a confirmed subscription")

	evListener.SubscriberRegistered(fullServiceName, "")
	assert.True(t, evListener.IsSubscriberRegistered(key))

	// re-caching a confirmed subscription must not lose the confirmed state
	evListener.CacheSubscriberForRedo(fullServiceName, "")
	assert.True(t, evListener.IsSubscriberRegistered(key))

	evListener.RemoveSubscriberForRedo(fullServiceName, "")
	assert.False(t, evListener.IsSubscriberCached(key))
	assert.False(t, evListener.IsSubscriberRegistered(key))

	// confirming an entry that was never cached is a no-op
	evListener.SubscriberRegistered(fullServiceName, "")
	assert.False(t, evListener.IsSubscriberRegistered(key))
}

// TestSubscriberRedoOnDisconnect covers the regression from the second review:
// a subscription confirmed on a lost connection must become unconfirmed on
// disconnect, and a failed reconnect redo must remain unconfirmed so later
// reads keep retrying it.
func TestSubscriberRedoOnDisconnect(t *testing.T) {
	evListener := NewConnectionEventListener(nil)
	fullServiceName := util.GetGroupName("service-a", "group-a")
	key := util.GetServiceCacheKey(fullServiceName, "")

	// confirmed subscription on the live connection
	evListener.CacheSubscriberForRedo(fullServiceName, "")
	evListener.SubscriberRegistered(fullServiceName, "")
	assert.True(t, evListener.IsSubscriberRegistered(key))

	// connection lost: the confirmed state must be reset while the redo intent survives
	evListener.OnDisConnect()
	assert.True(t, evListener.IsSubscriberCached(key), "the redo intent must survive the disconnect for reconnect redo")
	assert.False(t, evListener.IsSubscriberRegistered(key), "a subscription confirmed on the lost connection must no longer be reported as subscribed")

	// reconnect redo fails: the entry stays cached but unconfirmed
	evListener.CacheSubscriberForRedo(fullServiceName, "")
	assert.False(t, evListener.IsSubscriberRegistered(key), "a failed redo must remain unconfirmed so later reads retry")

	// redo eventually succeeds: confirmed again
	evListener.SubscriberRegistered(fullServiceName, "")
	assert.True(t, evListener.IsSubscriberRegistered(key))
}

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
