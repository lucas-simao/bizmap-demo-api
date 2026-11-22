package main

import (
	"net/http"

	"github.com/labstack/echo/v4"
)

type Handler struct {
	repo   *Repo
	events *Events
}

func NewHandler(repo *Repo, events *Events) *Handler { return &Handler{repo: repo, events: events} }

type signupReq struct {
	Email string `json:"email"`
	Name  string `json:"name"`
}

// CreateCustomer signs a shopper up.
func (h *Handler) CreateCustomer(c echo.Context) error {
	var req signupReq
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	id, err := h.repo.InsertCustomer(c.Request().Context(), req.Email, req.Name)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusCreated, map[string]string{"id": id})
}

// GetCustomer returns a customer profile.
func (h *Handler) GetCustomer(c echo.Context) error {
	cus, err := h.repo.GetCustomer(c.Request().Context(), c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound)
	}
	return c.JSON(http.StatusOK, cus)
}

// ListProducts returns the catalog.
func (h *Handler) ListProducts(c echo.Context) error {
	ps, err := h.repo.ListProducts(c.Request().Context())
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, ps)
}

type orderReq struct {
	CustomerID string `json:"customer_id"`
	Lines      []Line `json:"lines"`
}

// CreateOrder prices the cart (discounts), reserves stock and opens the order.
func (h *Handler) CreateOrder(c echo.Context) error {
	ctx := c.Request().Context()
	var req orderReq
	if err := c.Bind(&req); err != nil || len(req.Lines) == 0 {
		return echo.NewHTTPError(http.StatusBadRequest, "lines required")
	}
	cus, err := h.repo.GetCustomer(ctx, req.CustomerID)
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "customer not found")
	}
	o, err := h.repo.PlaceOrder(ctx, cus, req.Lines)
	if err != nil {
		return echo.NewHTTPError(http.StatusConflict, err.Error())
	}
	_ = h.events.Publish(ctx, o.ID, "order_placed", o.Status)
	return c.JSON(http.StatusCreated, o)
}

// GetOrder returns one order.
func (h *Handler) GetOrder(c echo.Context) error {
	o, err := h.repo.GetOrder(c.Request().Context(), c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound)
	}
	return c.JSON(http.StatusOK, o)
}

type payReq struct {
	ProviderRef string `json:"provider_ref"`
}

// PayOrder records the payment of a pending order.
func (h *Handler) PayOrder(c echo.Context) error {
	ctx := c.Request().Context()
	var req payReq
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	o, err := h.repo.GetOrder(ctx, c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound)
	}
	if err := CanTransition(o.Status, "paid"); err != nil {
		return echo.NewHTTPError(http.StatusConflict, err.Error())
	}
	if err := h.repo.RecordPayment(ctx, o.ID, req.ProviderRef, o.TotalCents); err != nil {
		return err
	}
	_ = h.events.Publish(ctx, o.ID, "order_paid", "paid")
	return c.NoContent(http.StatusNoContent)
}

type shipReq struct {
	Carrier  string `json:"carrier"`
	Tracking string `json:"tracking_code"`
}

// ShipOrder hands a paid order to the carrier.
func (h *Handler) ShipOrder(c echo.Context) error {
	ctx := c.Request().Context()
	var req shipReq
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	o, err := h.repo.GetOrder(ctx, c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound)
	}
	if err := CanTransition(o.Status, "shipped"); err != nil {
		return echo.NewHTTPError(http.StatusConflict, err.Error())
	}
	if err := h.repo.RecordShipment(ctx, o.ID, req.Carrier, req.Tracking); err != nil {
		return err
	}
	_ = h.events.Publish(ctx, o.ID, "order_shipped", "shipped")
	return c.NoContent(http.StatusNoContent)
}

// CancelOrder cancels a pending or paid order and releases its stock.
func (h *Handler) CancelOrder(c echo.Context) error {
	ctx := c.Request().Context()
	o, err := h.repo.GetOrder(ctx, c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound)
	}
	if err := CanTransition(o.Status, "cancelled"); err != nil {
		return echo.NewHTTPError(http.StatusConflict, err.Error())
	}
	if err := h.repo.CancelOrder(ctx, o.ID); err != nil {
		return err
	}
	_ = h.events.Publish(ctx, o.ID, "order_cancelled", "cancelled")
	return c.NoContent(http.StatusNoContent)
}
