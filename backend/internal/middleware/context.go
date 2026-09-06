package middleware

import (
	"context"

	"github.com/047pegasus/go-boilerplate/internal/apm"
	loggerPkg "github.com/047pegasus/go-boilerplate/internal/logger"
	"github.com/047pegasus/go-boilerplate/internal/server"
	"github.com/labstack/echo/v5"
	"github.com/rs/zerolog"
)

const (
	UserIDKey   = "user_id"
	UserRoleKey = "user_role"
	LoggerKey   = "logger"
)

type ContextEnhancer struct {
	server *server.Server
}

func NewContextEnhancer(server *server.Server) *ContextEnhancer {
	return &ContextEnhancer{
		server: server,
	}
}
func (ctxEnhance *ContextEnhancer) EnhanceContext() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			//Extract RequestID
			requestID := GetRequestID(c)
			//CREATE ENHANCEMENT IN CTX FROM REQUEST CONTEXT
			contextLogger := ctxEnhance.server.Logger.With().
				Str("request_id", requestID).
				Str("method", c.Request().Method).
				Str("path", c.Request().URL.Path).
				Str("remote_addr", c.Request().RemoteAddr).
				Str("user_agent", c.Request().UserAgent()).
				Str("ip", c.RealIP()).
				Logger()

			//Add trace ctx, if available (which means if Sentry is enabled and available)
			if txn := apm.FromContext(c.Request().Context()); txn != nil {
				contextLogger = loggerPkg.WithTraceContext(contextLogger, txn.Span())
			}
			if UserID := ctxEnhance.extractUserID(c); UserID != "" {
				contextLogger = contextLogger.With().Str("user_id", UserID).Logger()
			}
			if userRole := ctxEnhance.extractUserRole(c); userRole != "" {
				contextLogger = contextLogger.With().Str("user_role", userRole).Logger()
			}

			c.Set(LoggerKey, &contextLogger)
			ctx := context.WithValue(c.Request().Context(), LoggerKey, &contextLogger)
			c.SetRequest(c.Request().WithContext(ctx))

			return next(c)
		}
	}
}

func (ctxEnhance *ContextEnhancer) extractUserID(c *echo.Context) string {
	if userID, ok := c.Request().Context().Value(UserIDKey).(string); ok {
		return userID
	}
	return ""
}

func (ctxEnhance *ContextEnhancer) extractUserRole(c *echo.Context) string {
	if userRole, ok := c.Request().Context().Value(UserRoleKey).(string); ok {
		return userRole
	}
	return ""
}
func GetUserID(c *echo.Context) string {
	if userID, ok := c.Get(UserIDKey).(string); ok {
		return userID
	}
	return ""
}

func GetLogger(c *echo.Context) *zerolog.Logger {
	if logger, ok := c.Get(LoggerKey).(*zerolog.Logger); ok {
		return logger
	}
	//cant return nil need to fallback to a basic logger if the logger is not found
	basicLogger := zerolog.Nop()
	return &basicLogger
}
