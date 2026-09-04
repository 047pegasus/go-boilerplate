package database

import (
	"context"

	"github.com/getsentry/sentry-go"
	"github.com/jackc/pgx/v5"
)

// SentryTracer implements pgx.QueryTracer using Sentry performance spans.
// It's a drop-in replacement for nrpgx5.NewTracer() — pass the return value
// of NewSentryTracer() to pgxPoolConfig.ConnConfig.Tracer the same way you
// would nrpgx5.NewTracer().
type SentryTracer struct{}

// NewSentryTracer creates a new value which implements pgx.QueryTracer.
func NewSentryTracer() *SentryTracer {
	return &SentryTracer{}
}

type sentrySpanCtxKey struct{}

// TraceQueryStart is called by pgx/v5 at the beginning of Query, QueryRow,
// and Exec calls. It starts a Sentry span and stashes it on the context so
// TraceQueryEnd can finish it.
func (t *SentryTracer) TraceQueryStart(
	ctx context.Context,
	conn *pgx.Conn,
	data pgx.TraceQueryStartData,
) context.Context {
	span := sentry.StartSpan(ctx, "db.sql.query",
		sentry.WithDescription(data.SQL),
	)
	span.SetData("db.system", "postgresql")

	return context.WithValue(span.Context(), sentrySpanCtxKey{}, span)
}

// TraceQueryEnd is called by pgx/v5 at the end of Query, QueryRow, and Exec
// calls. It finishes the Sentry span started in TraceQueryStart.
func (t *SentryTracer) TraceQueryEnd(
	ctx context.Context,
	conn *pgx.Conn,
	data pgx.TraceQueryEndData,
) {
	span, ok := ctx.Value(sentrySpanCtxKey{}).(*sentry.Span)
	if !ok || span == nil {
		return
	}
	if data.Err != nil {
		span.Status = sentry.SpanStatusInternalError
		span.SetData("error", data.Err.Error())
	} else {
		span.Status = sentry.SpanStatusOK
	}
	span.Finish()
}
