package http_agent

import (
	"github.com/nacos-group/nacos-sdk-go/v2/common/monitor"
	"github.com/nacos-group/nacos-sdk-go/v2/util"
	"net/http"
	"time"
)

type MetricHttpAgent struct {
	httpAgent *HttpAgent
}

func NewMetricHttpAgent(agentProxy *HttpAgent) *MetricHttpAgent {
	return &MetricHttpAgent{
		httpAgent: agentProxy,
	}
}

var defaultMonitorCode = "NA"

func (agent *MetricHttpAgent) Get(path string, header http.Header, timeoutMs uint64, params map[string]string) (response *http.Response, err error) {
	start := time.Now()
	response, err = agent.httpAgent.Get(path, header, timeoutMs, params)
	monitor.GetConfigRequestMonitor("GET", path, util.GetStatusCode(response)).Observe(float64(time.Now().Nanosecond() - start.Nanosecond()))
	return
}
func (agent *MetricHttpAgent) Post(path string, header http.Header, timeoutMs uint64, params map[string]string) (response *http.Response, err error) {
	start := time.Now()
	response, err = agent.httpAgent.Post(path, header, timeoutMs, params)
	monitor.GetConfigRequestMonitor("POST", path, util.GetStatusCode(response)).Observe(float64(time.Now().Nanosecond() - start.Nanosecond()))
	return
}
func (agent *MetricHttpAgent) Delete(path string, header http.Header, timeoutMs uint64, params map[string]string) (response *http.Response, err error) {
	start := time.Now()
	response, err = agent.httpAgent.Delete(path, header, timeoutMs, params)
	monitor.GetConfigRequestMonitor("DELETE", path, util.GetStatusCode(response)).Observe(float64(time.Now().Nanosecond() - start.Nanosecond()))
	return
}
func (agent *MetricHttpAgent) Put(path string, header http.Header, timeoutMs uint64, params map[string]string) (response *http.Response, err error) {
	start := time.Now()
	response, err = agent.httpAgent.Put(path, header, timeoutMs, params)
	monitor.GetConfigRequestMonitor("PUT", path, util.GetStatusCode(response)).Observe(float64(time.Now().Nanosecond() - start.Nanosecond()))
	return
}
func (agent *MetricHttpAgent) RequestOnlyResult(method string, path string, header http.Header, timeoutMs uint64, params map[string]string) string {
	start := time.Now()
	result := agent.httpAgent.RequestOnlyResult(method, path, header, timeoutMs, params)
	monitor.GetConfigRequestMonitor(method, path, defaultMonitorCode).Observe(float64(time.Now().Nanosecond() - start.Nanosecond()))
	return result
}
func (agent *MetricHttpAgent) Request(method string, path string, header http.Header, timeoutMs uint64, params map[string]string) (response *http.Response, err error) {
	start := time.Now()
	response, err = agent.httpAgent.Request(method, path, header, timeoutMs, params)
	monitor.GetConfigRequestMonitor(method, path, defaultMonitorCode).Observe(float64(time.Now().Nanosecond() - start.Nanosecond()))
	return
}
