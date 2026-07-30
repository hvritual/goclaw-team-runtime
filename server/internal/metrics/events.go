package metrics

import "github.com/multica-ai/multica/server/internal/analytics"

// BusinessMetrics is retained as a narrow extension point for deployments
// that add operational counters outside the six-domain application core.
type BusinessMetrics struct{}

// RecordEvent forwards non-metrics-only events to the configured analytics
// client. Domain handlers can call it without depending on a metrics backend.
func RecordEvent(client analytics.Client, _ *BusinessMetrics, event analytics.Event) {
	if client != nil && !analytics.IsMetricsOnly(event.Name) {
		client.Capture(event)
	}
}
