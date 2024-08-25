package server

import (
	"net/http"

	"github.com/firestarter2501/prom-apcupsd-exporter/pkg/metric"

	"github.com/go-kit/kit/log/level"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var metricsPromHandler = promhttp.Handler()

// metricsInit ..
func metricsInit() {
	http.HandleFunc("/metrics", metrcisHandle)
}

func metrcisHandle(w http.ResponseWriter, r *http.Request) {
	onComplete := make(chan bool)
	collector.Collect(metric.CollectOpts{
		PreventFlood: true,
		OnComplete:   onComplete,
	})
	if <-onComplete {
		level.Debug(logger).Log("msg", "metrcisHandle begin")
		metricsPromHandler.ServeHTTP(w, r)
		level.Debug(logger).Log("msg", "metrcisHandle end")
	}
}
