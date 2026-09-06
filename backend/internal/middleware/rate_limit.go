package middleware

import (
	"github.com/047pegasus/go-boilerplate/internal/server"
	"github.com/getsentry/sentry-go"
)

type RateLimitMiddleware struct {
	server *server.Server
}

func NewRateLimiter(server *server.Server) *RateLimitMiddleware {
	return &RateLimitMiddleware{server: server}
}
func (rl *RateLimitMiddleware) RecordRateLimitHit(endpoint string) {
	client := rl.server.LoggerService.GetApplication()
	if rl.server.LoggerService == nil || client == nil {
		return
	}
	event := sentry.NewEvent()
	event.Message = "RateLimitHit"
	event.Level = sentry.LevelWarning
	event.Tags = map[string]string{
		"event_type": "RateLimitHit",
		"endpoint":   endpoint,
	}
	client.CaptureEvent(event, nil, sentry.NewScope())
}
