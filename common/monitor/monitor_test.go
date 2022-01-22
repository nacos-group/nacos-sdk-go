package monitor

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestGaugeMonitor(t *testing.T) {
	t.Run("getGaugeWithLabels", func(t *testing.T) {
		// will panic because of wrong label count.
		defer func() {
			r := recover()
			assert.NotNil(t, r)
		}()
		GetGaugeWithLabels("gauge", "test_gauge", "should_not_exist_label")
	})

	t.Run("serviceInfoMapMonitor", func(t *testing.T) {
		monitor := GetServiceInfoMapSizeMonitor()
		assert.NotNil(t, monitor)
	})

	t.Run("getHistogram", func(t *testing.T) {
		// will panic because of wrong label count.
		defer func() {
			r := recover()
			assert.NotNil(t, r)
		}()
		GetHistogramWithLabels("histogram", "test_histogram", "should_not_exist_label")
	})

	t.Run("serviceInfoMapMonitor", func(t *testing.T) {
		monitor := GetConfigRequestMonitor("GET", "url", "NA")
		assert.NotNil(t, monitor)
	})

}
