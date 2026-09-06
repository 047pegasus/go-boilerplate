package router

import (
	"github.com/047pegasus/go-boilerplate/internal/handler"
	"github.com/labstack/echo/v5"
)

func registerSystemRoutes(r *echo.Echo, h *handler.Handlers) {
	r.GET("/status", h.Health.CheckHealth)
	r.Static("/static", "static")
	r.GET("/docs", h.OpenAPI.ServeOpenAPIUI)
}
