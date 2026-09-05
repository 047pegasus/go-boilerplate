package custom

import (
	"context"
	"time"

	"github.com/getsentry/sentry-go"
	"github.com/valkey-io/valkey-go"
	"github.com/valkey-io/valkey-go/valkeyhook"
)

// sentryValkeyHook implements valkeyhook.Hook using Sentry performance spans.
// It's a drop-in replacement for nrredis.NewHook() — wrap your client with
// WithSentryHook(client) the same way you'd call client.AddHook(nrredis.NewHook(...)).
type sentryValkeyHook struct{}

// WithSentryHook wraps a valkey.Client so every command is traced as a
// Sentry span. Mirrors redisClient.AddHook(nrredis.NewHook(...)).
func WithSentryHook(client valkey.Client) valkey.Client {
	return valkeyhook.WithHook(client, &sentryValkeyHook{})
}

func startValkeySpan(ctx context.Context, op string) *sentry.Span {
	span := sentry.StartSpan(ctx, op)
	span.SetData("db.system", "valkey")
	return span
}

func finishValkeySpan(span *sentry.Span, err error) {
	if err != nil {
		span.Status = sentry.SpanStatusInternalError
		span.SetData("error", err.Error())
	} else {
		span.Status = sentry.SpanStatusOK
	}
	span.Finish()
}

func (h *sentryValkeyHook) Do(client valkey.Client, ctx context.Context, cmd valkey.Completed) (resp valkey.ValkeyResult) {
	span := startValkeySpan(ctx, "cache.valkey.do")
	resp = client.Do(span.Context(), cmd)
	finishValkeySpan(span, resp.Error())
	return resp
}

func (h *sentryValkeyHook) DoMulti(client valkey.Client, ctx context.Context, multi ...valkey.Completed) (resps []valkey.ValkeyResult) {
	span := startValkeySpan(ctx, "cache.valkey.do_multi")
	resps = client.DoMulti(span.Context(), multi...)

	var firstErr error
	for _, r := range resps {
		if err := r.Error(); err != nil {
			firstErr = err
			break
		}
	}
	finishValkeySpan(span, firstErr)
	return resps
}

func (h *sentryValkeyHook) DoCache(client valkey.Client, ctx context.Context, cmd valkey.Cacheable, ttl time.Duration) (resp valkey.ValkeyResult) {
	span := startValkeySpan(ctx, "cache.valkey.do_cache")
	resp = client.DoCache(span.Context(), cmd, ttl)
	finishValkeySpan(span, resp.Error())
	return resp
}

func (h *sentryValkeyHook) DoMultiCache(client valkey.Client, ctx context.Context, multi ...valkey.CacheableTTL) (resps []valkey.ValkeyResult) {
	span := startValkeySpan(ctx, "cache.valkey.do_multi_cache")
	resps = client.DoMultiCache(span.Context(), multi...)

	var firstErr error
	for _, r := range resps {
		if err := r.Error(); err != nil {
			firstErr = err
			break
		}
	}
	finishValkeySpan(span, firstErr)
	return resps
}

func (h *sentryValkeyHook) Receive(client valkey.Client, ctx context.Context, subscribe valkey.Completed, fn func(msg valkey.PubSubMessage)) (err error) {
	span := startValkeySpan(ctx, "cache.valkey.receive")
	err = client.Receive(span.Context(), subscribe, fn)
	finishValkeySpan(span, err)
	return err
}

func (h *sentryValkeyHook) DoStream(client valkey.Client, ctx context.Context, cmd valkey.Completed) valkey.ValkeyResultStream {
	span := startValkeySpan(ctx, "cache.valkey.do_stream")
	stream := client.DoStream(span.Context(), cmd)
	// Streams don't resolve synchronously, so there's no single error to
	// attach here — finish the span once the stream is handed back.
	span.Finish()
	return stream
}

func (h *sentryValkeyHook) DoMultiStream(client valkey.Client, ctx context.Context, multi ...valkey.Completed) valkey.MultiValkeyResultStream {
	span := startValkeySpan(ctx, "cache.valkey.do_multi_stream")
	stream := client.DoMultiStream(span.Context(), multi...)
	span.Finish()
	return stream
}
