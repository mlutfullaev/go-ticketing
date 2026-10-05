package book

import (
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
	g.POST("", h.Book, protected([]string{"user", "admin"}))
}

func (h *Handler) Book(c *echo.Context) error {
	ctx := c.Request().Context()
	var body CreateHold

	if err := c.Bind(&body); err != nil {
		return c.JSON(http.StatusBadRequest, response.BadRequest(err.Error()))
	}

	if err := c.Validate(&body); err != nil {
		return c.JSON(http.StatusBadRequest, response.BadRequest(err.Error()))
	}

	user, ok := c.Get("userID").(uint64)
	if !ok || user == 0 {
		return c.JSON(http.StatusUnauthorized, response.Unauthorized("Unauthorized"))
	}

	hold, err := h.service.Book(ctx, &body, uint(user))
	if err != nil {
		return c.JSON(http.StatusInternalServerError, response.BadRequest(err.Error()))
	}

	return c.JSON(http.StatusOK, response.Ok(hold))
}
