/*
 * Copyright 1999-2020 Alibaba Group Holding Ltd.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *      http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package monitor

import "github.com/prometheus/client_golang/prometheus"

var (
	gaugeMonitorVec = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "nacos_monitor",
		Help: "nacos_monitor",
	}, []string{"module", "name"})
	histogramMonitorVec = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name: "nacos_client_request",
		Help: "nacos_client_request",
	}, []string{"module", "method", "url", "code"})
)

// register collectors vec
func init() {
	prometheus.MustRegister(gaugeMonitorVec, histogramMonitorVec)
}

// get gauge with labels and use gaugeMonitorVec
func GetGaugeWithLabels(labels ...string) prometheus.Gauge {
	return gaugeMonitorVec.WithLabelValues(labels...)
}

func GetServiceInfoMapSizeMonitor() prometheus.Gauge {
	return GetGaugeWithLabels("serviceInfo", "serviceInfoMapSize")
}

func GetDom2BeatSizeMonitor() prometheus.Gauge {
	return GetGaugeWithLabels("dom2Beat", "dom2BeatSize")
}

func GetListenConfigCountMonitor() prometheus.Gauge {
	return GetGaugeWithLabels("listenConfig", "listenConfigCount")
}

// get histogram with labels and use histogramMonitorVec
func GetHistogramWithLabels(labels ...string) prometheus.Observer {
	return histogramMonitorVec.WithLabelValues(labels...)
}

func GetConfigRequestMonitor(method, url, code string) prometheus.Observer {
	return GetHistogramWithLabels("config", method, url, code)
}

func GetNamingRequestMonitor(method, url, code string) prometheus.Observer {
	return GetHistogramWithLabels("naming", method, url, code)
}
