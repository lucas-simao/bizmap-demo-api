package main

import "github.com/labstack/echo/v4"

// registerRoutes mounts every endpoint under the group it receives (/api).
func registerRoutes(g *echo.Group, h *Handler) {
	customers := g.Group("/customers")
	customers.POST("", h.CreateCustomer)
	customers.GET("/:id", h.GetCustomer)

	g.GET("/products", h.ListProducts)

	orders := g.Group("/orders")
	orders.POST("", h.CreateOrder)
	orders.GET("/:id", h.GetOrder)
	orders.POST("/:id/pay", h.PayOrder)
	orders.POST("/:id/ship", h.ShipOrder)
	orders.POST("/:id/cancel", h.CancelOrder)
}
