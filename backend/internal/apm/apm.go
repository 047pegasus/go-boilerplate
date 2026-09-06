package apm

import (
	"context"

	"github.com/getsentry/sentry-go"
)

// txnContextKey is used to store the current Sentry span on a plain
// context.Context. sentryecho only attaches the span to echo.Context, so
// TracingMiddleware.EnhanceTracing (see middleware/tracing.go) is
// responsible for copying it in here — mirroring what nrecho did automatically.
type txnContextKey struct{}

// Transaction wraps *sentry.Span, giving call sites the same shape they used
// with *newrelic.Transaction.
type Transaction struct {
	span *sentry.Span
}

// NewTransaction wraps a *sentry.Span. Returns nil if span is nil, so
// downstream nil-checks (`if txn != nil`) keep working unchanged.
func NewTransaction(span *sentry.Span) *Transaction {
	if span == nil {
		return nil
	}
	return &Transaction{span: span}
}

// ContextWithTransaction stores txn on ctx for later retrieval via FromContext.
func ContextWithTransaction(ctx context.Context, txn *Transaction) context.Context {
	if txn == nil {
		return ctx
	}
	return context.WithValue(ctx, txnContextKey{}, txn)
}

// FromContext is a drop-in replacement for newrelic.FromContext(ctx).
func FromContext(ctx context.Context) *Transaction {
	txn, _ := ctx.Value(txnContextKey{}).(*Transaction)
	return txn
}

// AddAttribute mirrors (*newrelic.Transaction).AddAttribute.
func (t *Transaction) AddAttribute(key string, value interface{}) {
	if t == nil || t.span == nil {
		return
	}
	t.span.SetData(key, value)
}

// NoticeError mirrors (*newrelic.Transaction).NoticeError, reporting err to
// the hub tied to this transaction's request when one is available.
func (t *Transaction) NoticeError(err error) {
	if err == nil {
		return
	}
	if t == nil || t.span == nil {
		sentry.CaptureException(err)
		return
	}
	if hub := sentry.GetHubFromContext(t.span.Context()); hub != nil {
		hub.CaptureException(err)
		return
	}
	sentry.CaptureException(err)
}

// WrapError mirrors nrpkgerrors.Wrap(err). Sentry captures stack traces
// itself via runtime.Callers, so this is a no-op — it exists purely so call
// sites keep the same shape instead of needing the wrapper call deleted.
func WrapError(err error) error {
	return err
}

// Span exposes the underlying *sentry.Span, e.g. for logger.WithTraceContext.
func (t *Transaction) Span() *sentry.Span {
	if t == nil {
		return nil
	}
	return t.span
}
