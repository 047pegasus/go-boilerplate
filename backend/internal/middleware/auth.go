package middleware

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/047pegasus/go-boilerplate/internal/errs"
	"github.com/047pegasus/go-boilerplate/internal/server"
	"github.com/clerk/clerk-sdk-go/v2"
	clerkhttp "github.com/clerk/clerk-sdk-go/v2/http"
	"github.com/labstack/echo/v5"
)

type AuthMiddleware struct {
	server *server.Server
}

func NewAuthMiddleware(s *server.Server) *AuthMiddleware {
	return &AuthMiddleware{server: s}
}

// Creating RequireAuth dependency which can be injected later for auth checks (uses Clerk)
func (auth *AuthMiddleware) RequireAuth(next echo.HandlerFunc) echo.HandlerFunc {
	return echo.WrapMiddleware(clerkhttp.WithHeaderAuthorization(
		clerkhttp.AuthorizationFailureHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			w.Header().Set("content-type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)

			response := map[string]string{
				"code":     "UNAUTHORIZED",
				"msg":      "Unauthorized",
				"override": "false",
				"status":   "401",
			}

			if err := json.NewEncoder(w).Encode(response); err != nil {
				auth.server.Logger.Error().Err(err).Str("method", "RequireAuth").Dur("duration", time.Since(start)).Msg("failed to encode JSON response")
			} else {
				auth.server.Logger.Error().Str("method", "RequireAuth").Dur("duration", time.Since(start)).Msg("could not get sessions claims from ctx")
			}
		}))))(func(c *echo.Context) error {
		start := time.Now()
		claims, ok := clerk.SessionClaimsFromContext(c.Request().Context())
		if !ok {
			auth.server.Logger.Error().
				Str("method", "RequireAuth").
				Str("request_id", GetRequestID(c)).
				Dur("duration", time.Since(start)).
				Msg("could not get session claims from ctx")
			return errs.NewUnAuthorizedError("Unauthorized", false)
		}
		c.Set("user_id", claims.Subject)
		c.Set("user_role", claims.ActiveOrganizationRole)
		c.Set("permissions", claims.Claims.ActiveOrganizationPermissions)

		auth.server.Logger.Info().
			Str("method", "RequireAuth").
			Str("user_id", claims.Subject).
			Str("request_id", GetRequestID(c)).
			Dur("duration", time.Since(start)).
			Msg("User authenticated successfully, request authorized")

		return next(c)
	})
}
