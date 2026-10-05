package event

import (
	"net/http"
	"strconv"
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
	g.POST("", h.CreateEvent, protected([]string{"admin"}))
	g.GET("", h.GetEvents)
	g.GET("/:id", h.GetEventByID)
}

func (h *Handler) CreateEvent(c *echo.Context) error {
	var createEvent CreateEvent

	if err := c.Bind(&createEvent); err != nil {
		return c.JSON(http.StatusBadRequest, response.BadRequest(err.Error()))
	}

	if err := c.Validate(&createEvent); err != nil {
		return c.JSON(http.StatusBadRequest, response.BadRequest(err.Error()))
	}

	event, err := h.service.CreateEvent(&createEvent)
	if err != nil {
		return c.JSON(http.StatusBadRequest, response.BadRequest(err.Error()))
	}

	return c.JSON(http.StatusCreated, response.Created(event))
}

func (h *Handler) GetEvents(c *echo.Context) error {
	events, err := h.service.GetEvents()
	if err != nil {
		return c.JSON(http.StatusBadRequest, response.BadRequest(err.Error()))
	}

	return c.JSON(http.StatusOK, events)
}

func (h *Handler) GetEventByID(c *echo.Context) error {
	ctx := c.Request().Context()
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		return c.JSON(http.StatusBadRequest, response.BadRequest(err.Error()))
	}

	event, err := h.service.GetEventByID(ctx, uint(id))
	if err != nil {
		return c.JSON(http.StatusBadRequest, response.BadRequest(err.Error()))
	}

	return c.JSON(http.StatusOK, response.Ok(event))
}
