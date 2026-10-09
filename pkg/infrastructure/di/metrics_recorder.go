package di

import (
	ctrlmetrics "sigs.k8s.io/controller-runtime/pkg/metrics"

	"github.com/scality/metalk8s-registry-node-agent/pkg/infrastructure/metricsrecorder"
	"github.com/scality/metalk8s-registry-node-agent/pkg/service"
)

// GetMetricsRecorder registers the metrics on the controller-runtime registry,
// so they are exposed by the manager metrics server.
func (c *Container) GetMetricsRecorder() service.MetricsRecorder {
	if c.metricsRecorder == nil {
		c.metricsRecorder = metricsrecorder.NewPrometheus(ctrlmetrics.Registry)
	}
	return c.metricsRecorder
}
