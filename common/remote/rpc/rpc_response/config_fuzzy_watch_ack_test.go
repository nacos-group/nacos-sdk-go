package rpc_response

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConfigFuzzyWatchAckResponseTypes(t *testing.T) {
	assert.Equal(t, "ConfigFuzzyWatchSyncResponse", (&ConfigFuzzyWatchSyncResponse{}).GetResponseType())
	assert.Equal(t, "ConfigFuzzyWatchChangeNotifyResponse", (&ConfigFuzzyWatchChangeNotifyResponse{}).GetResponseType())
}
