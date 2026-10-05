package order

import (
	"context"
	"net/http"
	"ticketing/pkg/response"

	"github.com/labstack/echo/v5"
)

type Handler struct {
	service Service
}

func NewHandler(service Service) *Handler {
	return &Handler{service}
}

func (h *Handler) RegisterRoutes(g *echo.Group, protected func(roles []string) echo.MiddlewareFunc) {
	g.POST("", h.CreateOrder, protected([]string{"user", "admin"}))
}

func (h *Handler) CreateOrder(c *echo.Context) error {
	var body CreateOrder
	if err := c.Bind(&body); err != nil {
		return c.JSON(http.StatusBadRequest, response.BadRequest(err.Error()))
	}

	if err := c.Validate(&body); err != nil {
		return c.JSON(http.StatusBadRequest, response.BadRequest(err.Error()))
	}

	userID, ok := c.Get("userID").(uint64)
	if !ok {
		return c.JSON(http.StatusBadRequest, response.BadRequest("user id not found"))
	}

	order, err := h.service.CreateOrder(context.Background(), &body, uint(userID))
	if err != nil {
		return c.JSON(http.StatusBadRequest, response.BadRequest(err.Error()))
	}

	return c.JSON(http.StatusCreated, response.Created(order))
}
