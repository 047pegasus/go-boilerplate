package apm

import (
	"context"

	"github.com/getsentry/sentry-go"
)

var metricsEnabled bool

// InitMetrics enables or disables the Count/Gauge/Distribution helpers
// below. Call this once at startup after the Sentry client is bound.
func InitMetrics(enabled bool) {
	metricsEnabled = enabled
}

// withFieldAttrs converts a field map into a WithAttributes MeterOption, so
// it can be prepended alongside any explicitly passed opts (e.g. WithUnit).
func withFieldAttrs(fields map[string]interface{}, opts []sentry.MeterOption) []sentry.MeterOption {
	if attrs := toAttributes(fields); len(attrs) > 0 {
		return append([]sentry.MeterOption{sentry.WithAttributes(attrs...)}, opts...)
	}
	return opts
}

// Count records an incrementing counter (e.g. requests handled, jobs
// completed), linked to ctx's current trace/span when present.
//
// Example:
//
//	apm.Count(c.Request().Context(), "rate_limit.hit", 1, map[string]interface{}{
//		"endpoint": c.Path(),
//	})
func Count(ctx context.Context, name string, value int64, fields map[string]interface{}, opts ...sentry.MeterOption) {
	if !metricsEnabled {
		return
	}
	sentry.NewMeter(ctx).Count(name, value, withFieldAttrs(fields, opts)...)
}

// Gauge records a point-in-time value that can go up or down (e.g. queue
// depth, active connections, pool size).
func Gauge(ctx context.Context, name string, value float64, fields map[string]interface{}, opts ...sentry.MeterOption) {
	if !metricsEnabled {
		return
	}
	sentry.NewMeter(ctx).Gauge(name, value, withFieldAttrs(fields, opts)...)
}

// Distribution records a value for statistical aggregation (e.g. response
// time, payload size). Pass sentry.WithUnit("millisecond") etc. via opts.
//
// Example:
//
//	apm.Distribution(ctx, "health_check.duration", float64(elapsed.Milliseconds()), nil,
//		sentry.WithUnit("millisecond"),
//	)
func Distribution(ctx context.Context, name string, value float64, fields map[string]interface{}, opts ...sentry.MeterOption) {
	if !metricsEnabled {
		return
	}
	sentry.NewMeter(ctx).Distribution(name, value, withFieldAttrs(fields, opts)...)
}
