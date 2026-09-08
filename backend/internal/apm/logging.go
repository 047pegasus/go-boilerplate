package apm

import (
	"context"
	"fmt"

	"github.com/getsentry/sentry-go"
	"github.com/getsentry/sentry-go/attribute"
)

var logsEnabled bool

// InitLogging enables or disables the Log* helpers below. Call this once at
// startup (see logger.NewLoggerService) after the Sentry client is bound.
func InitLogging(enabled bool) {
	logsEnabled = enabled
}

// toAttributes converts a plain field map into Sentry attributes, shared by
// both the logging and metrics helpers. The attribute API supports string,
// bool, int, and float64 — anything else (including int64, to stay
// conservative about what's actually available) is stringified via
// fmt.Sprintf rather than silently dropped.
func toAttributes(fields map[string]interface{}) []attribute.Builder {
	if len(fields) == 0 {
		return nil
	}
	attrs := make([]attribute.Builder, 0, len(fields))
	for k, v := range fields {
		switch val := v.(type) {
		case string:
			attrs = append(attrs, attribute.String(k, val))
		case bool:
			attrs = append(attrs, attribute.Bool(k, val))
		case int:
			attrs = append(attrs, attribute.Int(k, val))
		case int64:
			attrs = append(attrs, attribute.Int(k, int(val)))
		case float64:
			attrs = append(attrs, attribute.Float64(k, val))
		default:
			attrs = append(attrs, attribute.String(k, fmt.Sprintf("%v", val)))
		}
	}
	return attrs
}

// logAt is the shared implementation behind LogTrace/LogDebug/.../LogFatal.
// ctx is required — it's how Sentry links a log entry to the current
// request's trace/span, exactly like apm.FromContext does for transactions.
func logAt(ctx context.Context, chooseLevel func(sentry.Logger) sentry.LogEntry, msg string, fields map[string]interface{}) {
	if !logsEnabled {
		return
	}
	logger := sentry.NewLogger(ctx)
	if attrs := toAttributes(fields); len(attrs) > 0 {
		logger.SetAttributes(attrs...)
	}
	chooseLevel(logger).Emit(msg)
}

// LogTrace/LogDebug/LogInfo/LogWarn/LogError/LogFatal send a structured log
// entry to Sentry, linked to ctx's trace when present. fields may be nil.
//
// Example:
//
//	apm.LogInfo(c.Request().Context(), "order processed", map[string]interface{}{
//		"order_id":   order.ID,
//		"cart_value": order.Total,
//	})
func LogTrace(ctx context.Context, msg string, fields map[string]interface{}) {
	logAt(ctx, sentry.Logger.Trace, msg, fields)
}

func LogDebug(ctx context.Context, msg string, fields map[string]interface{}) {
	logAt(ctx, sentry.Logger.Debug, msg, fields)
}

func LogInfo(ctx context.Context, msg string, fields map[string]interface{}) {
	logAt(ctx, sentry.Logger.Info, msg, fields)
}

func LogWarn(ctx context.Context, msg string, fields map[string]interface{}) {
	logAt(ctx, sentry.Logger.Warn, msg, fields)
}

func LogError(ctx context.Context, msg string, fields map[string]interface{}) {
	logAt(ctx, sentry.Logger.Error, msg, fields)
}

func LogFatal(ctx context.Context, msg string, fields map[string]interface{}) {
	logAt(ctx, sentry.Logger.Fatal, msg, fields)
}
