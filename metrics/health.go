package metrics

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

var HealthGauge = prometheus.NewGaugeVec(
	prometheus.GaugeOpts{
		Name: "service_health_status",
		Help: "Health status of services (1 = healthy, 0 = unhealthy)",
	},
	[]string{"service"},
)

func init() {
	prometheus.MustRegister(HealthGauge)
}

func StartHealthUpdater(serviceName string, intervalSeconds int) {
	go func() {
		for {
			HealthGauge.WithLabelValues(serviceName).Set(1)
			time.Sleep(time.Duration(intervalSeconds) * time.Second)
		}
	}()
}
