package utils

import (
	ctrlmetrics "sigs.k8s.io/controller-runtime/pkg/metrics"
)

// MetricValue returns the value of the gauge or counter series of the given metric,
// matching exactly the given labels, from the controller-runtime registry.
// The boolean is false when the series does not exist.
func MetricValue(name string, labels map[string]string) (float64, bool) {
	families, err := ctrlmetrics.Registry.Gather()
	if err != nil {
		return 0, false
	}

	for _, family := range families {
		if family.GetName() != name {
			continue
		}
		for _, metric := range family.GetMetric() {
			if len(metric.GetLabel()) != len(labels) {
				continue
			}
			matches := true
			for _, label := range metric.GetLabel() {
				if labels[label.GetName()] != label.GetValue() {
					matches = false
					break
				}
			}
			if !matches {
				continue
			}
			if metric.GetGauge() != nil {
				return metric.GetGauge().GetValue(), true
			}
			return metric.GetCounter().GetValue(), true
		}
	}

	return 0, false
}
