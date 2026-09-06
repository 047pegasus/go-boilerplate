package middleware

import (
	"github.com/047pegasus/go-boilerplate/internal/apm"
	"github.com/047pegasus/go-boilerplate/internal/server"
	sentryecho "github.com/getsentry/sentry-go/echo"
	"github.com/labstack/echo/v5"
)

type TracingMiddleware struct {
	server *server.Server
}

func NewTracingMiddleware(server *server.Server) *TracingMiddleware {
	return &TracingMiddleware{server: server}
}

func (tm *TracingMiddleware) SentryMiddleware() echo.MiddlewareFunc {
	//check if sentry is enabled by checking if the logger instance of it exists and is not nil
	if tm.server.LoggerService != nil && tm.server.LoggerService.GetApplication() != nil {
		return sentryecho.New(sentryecho.Options{Repanic: true})
	}
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return next
	}
}

// EnhanceTracing adds custom attributes to Sentry transactions
func (tm *TracingMiddleware) EnhanceTracing() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			span := sentryecho.GetSpanFromContext(c)
			txn := apm.NewTransaction(span)

			if txn != nil {
				//Get the transaction from context
				ctx := apm.ContextWithTransaction(span.Context(), txn)
				c.SetRequest(c.Request().WithContext(ctx))

				// service.name and service.environment are already set in logger and in Sentry config
				txn.AddAttribute("http.real_ip", c.RealIP())
				txn.AddAttribute("http.user_agent", c.Request().UserAgent())

				// Add request ID if available
				if RequestID := GetRequestID(c); RequestID != "" {
					txn.AddAttribute("request.id", RequestID)
				}

				//Add user context if available
				if userID := c.Get("user_id"); userID != nil {
					if UserIDStr, ok := userID.(string); ok && UserIDStr != "" {
						txn.AddAttribute("user.id", UserIDStr)
					}
				}
			}

			//execute next handler
			err := next(c)
			if err != nil {
				txn.NoticeError(err)
			}
			if span != nil {
				span.SetData("http.status_code", c.Response().Header().Values("status"))
			}
			return err
		}
	}
}
