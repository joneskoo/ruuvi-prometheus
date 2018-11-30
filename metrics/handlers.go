// Copyright Joonas Kuorilehto 2018.

package metrics

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var handlers = http.NewServeMux()

func init() {
	handlers.HandleFunc("/", handleRoot)
	handlers.Handle("/metrics", promhttp.Handler())
}

const rootContent = `ruuvi-prometheus exporter
https://github.com/joneskoo/ruuvi-prometheus

/                This page
/metrics         Prometheus metrics endpoint
`

func handleRoot(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(rootContent))
}
