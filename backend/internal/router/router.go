package router

import (
	"net/http"

	"github.com/047pegasus/go-boilerplate/internal/handler"
	"github.com/047pegasus/go-boilerplate/internal/middleware"
	"github.com/047pegasus/go-boilerplate/internal/server"
	"github.com/047pegasus/go-boilerplate/internal/service"
	"github.com/labstack/echo/v5"
	echoMiddleware "github.com/labstack/echo/v5/middleware"
)

func NewRouter(s *server.Server, h *handler.Handlers, services *service.Services) *echo.Echo {

	middlewares := middleware.NewMiddlewares(s)
	router := echo.New()
	router.HTTPErrorHandler = middlewares.Global.GlobalErrorHandler

	router.Use(
		echoMiddleware.RateLimiterWithConfig(echoMiddleware.RateLimiterConfig{
			Store: echoMiddleware.NewRateLimiterMemoryStore(20),
			DenyHandler: func(c *echo.Context, identifier string, err error) error {
				//record rate limit hit metrics
				if rateLimitMiddleware := middlewares.RateLimit; rateLimitMiddleware != nil {
					rateLimitMiddleware.RecordRateLimitHit(c.Path())
				}

				s.Logger.Warn().
					Str("request_id", middleware.GetRequestID(c)).
					Str("identifier", identifier).
					Str("path", c.Path()).
					Str("method", c.Request().Method).
					Str("ip", c.RealIP()).
					Msg("rate limit exceeded")

				return echo.NewHTTPError(http.StatusTooManyRequests, "rate limit exceeded")
			},
		}),
		middlewares.Global.CORS(),
		middlewares.Global.Security(),
		middleware.RequestID(),
		middlewares.Tracing.SentryMiddleware(),
		middlewares.Tracing.EnhanceTracing(),
		middlewares.Global.RequestLogger(),
		middlewares.Global.Recover(),
	)

	//registering system level routes
	registerSystemRoutes(router, h)
	//registering versioned routes
	router.Group("/api/v1")

	return router
}
