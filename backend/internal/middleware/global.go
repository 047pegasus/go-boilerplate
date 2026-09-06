package middleware

import (
	"net/http"

	"github.com/047pegasus/go-boilerplate/internal/errs"
	"github.com/047pegasus/go-boilerplate/internal/server"
	"github.com/047pegasus/go-boilerplate/internal/sqlerr"
	"github.com/labstack/echo/v5"
	"github.com/labstack/echo/v5/middleware"
	"github.com/pkg/errors"
	"github.com/rs/zerolog"
)

type GlobalMiddlewares struct {
	server *server.Server
}

func NewGlobalMiddlewares(server *server.Server) *GlobalMiddlewares {
	return &GlobalMiddlewares{server: server}
}

// setup CORS global middleware
func (global *GlobalMiddlewares) CORS() echo.MiddlewareFunc {
	return middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins: global.server.Config.Server.CORSAllowedOrigins,
	})
}

// setup RequestLogger global middleware to log all incoming req
func (global *GlobalMiddlewares) RequestLogger() echo.MiddlewareFunc {
	return middleware.RequestLoggerWithConfig(middleware.RequestLoggerConfig{
		LogURI:      true,
		LogStatus:   true,
		HandleError: true,
		LogLatency:  true,
		LogHost:     true,
		LogMethod:   true,
		LogURIPath:  true,
		LogValuesFunc: func(c *echo.Context, v middleware.RequestLoggerValues) error {
			statusCode := v.Status
			if v.Error != nil {
				var httpErr *errs.HttpError
				var echoErr *echo.HTTPError
				if errors.As(v.Error, &httpErr) {
					statusCode = httpErr.StatusCode
				} else if errors.As(v.Error, &echoErr) {
					statusCode = echoErr.Code
				}
			}
			//get enhanced logger from context
			logger := GetLogger(c)
			var e *zerolog.Event
			switch {
			case statusCode >= 400 && statusCode < 500:
				e = logger.Warn()
			case statusCode >= 500:
				e = logger.Error().Err(v.Error)
			default:
				e = logger.Info()
			}

			//add request_id if available
			if requestID := GetRequestID(c); requestID != "" {
				e = e.Str("request_id", requestID)
			}
			//add user context if available
			if userID := GetUserID(c); userID != "" {
				e = e.Str("user_id", userID)
			}

			e.Dur("latency", v.Latency).
				Int("status", statusCode).
				Str("method", v.Method).
				Str("uri", v.URI).
				Str("host", v.Host).
				Str("ip", c.RealIP()).
				Str("user_agent", c.Request().UserAgent()).
				Msg("API")

			return nil
		},
	})
}

// Add panic recovery to make server instance auto recover from panics
func (global *GlobalMiddlewares) Recover() echo.MiddlewareFunc {
	return middleware.Recover()
}

// Add security middleware to inject security specific headers etc in the request
func (global *GlobalMiddlewares) Security() echo.MiddlewareFunc {
	return middleware.Secure()
}

// Now the global error handler
func (global *GlobalMiddlewares) GlobalErrorHandler(c *echo.Context, err error) {
	//Trying to handle db level errors here and converting them to appropriate http errors
	originalErr := err

	var httpErr *errs.HttpError
	if !errors.As(err, &httpErr) {
		var echoErr *echo.HTTPError
		if errors.As(err, &echoErr) {
			if echoErr.Code == http.StatusNotFound {
				err = errs.NewNotFoundError("Route not found", false, nil)
			}
		} else {
			// Here we call our sqlerr handler which will convert database errors
			// to appropriate application errors
			err = sqlerr.HandleError(err)
		}
	}

	//Processing converted err
	var echoErr *echo.HTTPError
	var status int
	var code string
	var message string
	var fieldErrors []errs.FieldError
	var action *errs.Action

	switch {
	case errors.As(err, &httpErr):
		status = httpErr.StatusCode
		code = httpErr.Code
		message = httpErr.Message
		fieldErrors = httpErr.Errors
		action = httpErr.Action
	case errors.As(err, &echoErr):
		status = echoErr.Code
		code = errs.MakeUpperCaseWithUnderscores(http.StatusText(status))
		if echoErr.Message != "" {
			message = echoErr.Message
		} else {
			message = http.StatusText(echoErr.Code)
		}
	default:
		status = http.StatusInternalServerError
		code = errs.MakeUpperCaseWithUnderscores(http.StatusText(http.StatusInternalServerError))
		message = http.StatusText(http.StatusInternalServerError)
	}

	//log original err for debug; use enhanced logger from context which already includes all fields
	logger := *GetLogger(c)
	logger.Error().Stack().
		Err(originalErr).
		Int("status", status).
		Str("error_code", code).
		Msg(message)

	resp, err := echo.UnwrapResponse(c.Response())

	if err != nil {
		logger.Error().Stack().Err(err).Int("status", status).Str("error_code", code).Msg(message)
	}
	if !resp.Committed {
		_ = c.JSON(status, errs.HttpError{
			Code:       code,
			Message:    message,
			StatusCode: status,
			Override:   httpErr != nil && httpErr.Override,
			Errors:     fieldErrors,
			Action:     action,
		})
	}
}
