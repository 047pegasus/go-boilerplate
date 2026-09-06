package handler

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/047pegasus/go-boilerplate/internal/middleware"
	"github.com/047pegasus/go-boilerplate/internal/server"
	"github.com/047pegasus/go-boilerplate/internal/server/custom/custom_utils"
	"github.com/getsentry/sentry-go"
	"github.com/labstack/echo/v5"
)

type HealthHandler struct {
	Handler
}

func NewHealthHandler(s *server.Server) *HealthHandler {
	return &HealthHandler{
		Handler: NewHandler(s),
	}
}

// recordHealthCheckError reports a health check failure to Sentry. Drop-in
// replacement for the New Relic RecordCustomEvent("HealthCheckError", ...)
// calls — captures a Sentry event with the same fields as tags/extra data.
func recordHealthCheckError(client *sentry.Client, fields map[string]interface{}) {
	if client == nil {
		return
	}

	tags := make(map[string]string, len(fields))
	for k, v := range fields {
		tags[k] = fmt.Sprintf("%v", v)
	}

	event := sentry.NewEvent()
	event.Message = "HealthCheckError"
	event.Level = sentry.LevelError
	event.Tags = tags
	event.Contexts = map[string]sentry.Context{
		"health_check": sentry.Context(fields),
	}

	client.CaptureEvent(event, nil, sentry.NewScope())
}

func (h *HealthHandler) CheckHealth(c *echo.Context) error {
	start := time.Now()
	logger := middleware.GetLogger(c).With().
		Str("operation", "health_check").
		Logger()

	response := map[string]interface{}{
		"status":      "healthy",
		"timestamp":   time.Now().UTC(),
		"environment": h.server.Config.Primary.Env,
		"checks":      make(map[string]interface{}),
	}

	checks := response["checks"].(map[string]interface{})
	isHealthy := true

	var sentryClient *sentry.Client
	if h.server.LoggerService != nil {
		sentryClient = h.server.LoggerService.GetApplication()
	}

	// Check database connectivity
	ctx, cancel := context.WithTimeout(c.Request().Context(), 5*time.Second)
	defer cancel()

	dbStart := time.Now()
	if err := h.server.DB.Pool.Ping(ctx); err != nil {
		checks["database"] = map[string]interface{}{
			"status":        "unhealthy",
			"response_time": time.Since(dbStart).String(),
			"error":         err.Error(),
		}
		isHealthy = false
		logger.Error().Err(err).Dur("response_time", time.Since(dbStart)).Msg("database health check failed")
		recordHealthCheckError(sentryClient, map[string]interface{}{
			"check_type":       "database",
			"operation":        "health_check",
			"error_type":       "database_unhealthy",
			"response_time_ms": time.Since(dbStart).Milliseconds(),
			"error_message":    err.Error(),
		})
	} else {
		checks["database"] = map[string]interface{}{
			"status":        "healthy",
			"response_time": time.Since(dbStart).String(),
		}
		logger.Info().Dur("response_time", time.Since(dbStart)).Msg("database health check passed")
	}

	// Database query spans are automatically captured by the Sentry pgx tracer

	// Check Valkey connectivity
	if h.server.Cache != nil {
		ctx, cancel := context.WithTimeout(c.Request().Context(), 5*time.Second)
		defer cancel()

		valkeyStart := time.Now()
		if _, err := custom_utils.PingValkeyBuildAndExecute(ctx, h.server.Cache); err != nil {
			checks["valkey"] = map[string]interface{}{
				"status":        "unhealthy",
				"response_time": time.Since(valkeyStart).String(),
				"error":         err.Error(),
			}
			logger.Error().Err(err).Dur("response_time", time.Since(valkeyStart)).Msg("valkey health check failed")
			recordHealthCheckError(sentryClient, map[string]interface{}{
				"check_type":       "valkey",
				"operation":        "health_check",
				"error_type":       "valkey_unhealthy",
				"response_time_ms": time.Since(valkeyStart).Milliseconds(),
				"error_message":    err.Error(),
			})
		} else {
			checks["valkey"] = map[string]interface{}{
				"status":        "healthy",
				"response_time": time.Since(valkeyStart).String(),
			}
			logger.Info().Dur("response_time", time.Since(valkeyStart)).Msg("valkey health check passed")
		}
	}

	// Set overall status
	if !isHealthy {
		response["status"] = "unhealthy"
		logger.Warn().
			Dur("total_duration", time.Since(start)).
			Msg("health check failed")
		recordHealthCheckError(sentryClient, map[string]interface{}{
			"check_type":        "overall",
			"operation":         "health_check",
			"error_type":        "overall_unhealthy",
			"total_duration_ms": time.Since(start).Milliseconds(),
		})
		return c.JSON(http.StatusServiceUnavailable, response)
	}

	logger.Info().
		Dur("total_duration", time.Since(start)).
		Msg("health check passed")

	err := c.JSON(http.StatusOK, response)
	if err != nil {
		logger.Error().Err(err).Msg("failed to write JSON response")
		recordHealthCheckError(sentryClient, map[string]interface{}{
			"check_type":    "response",
			"operation":     "health_check",
			"error_type":    "json_response_error",
			"error_message": err.Error(),
		})
		return fmt.Errorf("failed to write JSON response: %w", err)
	}

	return nil
}
