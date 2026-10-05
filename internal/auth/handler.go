package auth

import (
	"net/http"
	"ticketing/internal/user"
	"ticketing/pkg/response"

	"github.com/labstack/echo/v5"
)

type Handler struct {
	authService Service
}

func NewHandler(authService Service) *Handler {
	return &Handler{
		authService: authService,
	}
}

func (h *Handler) RegisterRoutes(g *echo.Group) {
	g.POST("/register", h.Register)
	g.POST("/login", h.Login)
	g.POST("/refresh", h.RefreshToken)
}

func (h *Handler) Register(c *echo.Context) error {
	var body user.CreateUser

	if err := c.Bind(&body); err != nil {
		return c.JSON(http.StatusBadRequest, response.BadRequest(err.Error()))
	}

	if err := c.Validate(&body); err != nil {
		return c.JSON(http.StatusBadRequest, response.BadRequest(err.Error()))
	}

	user, err := h.authService.Register(&body)
	if err != nil {
		return c.JSON(http.StatusBadRequest, response.BadRequest(err.Error()))
	}

	return c.JSON(http.StatusCreated, response.Created(user))
}

func (h *Handler) Login(c *echo.Context) error {
	var loginBody LoginBody

	if err := c.Bind(&loginBody); err != nil {
		return c.JSON(http.StatusBadRequest, response.BadRequest(err.Error()))
	}

	if err := c.Validate(&loginBody); err != nil {
		return c.JSON(http.StatusBadRequest, response.BadRequest(err.Error()))
	}

	user, err := h.authService.Login(&loginBody)

	if err != nil {
		return c.JSON(http.StatusBadRequest, response.BadRequest(err.Error()))
	}

	return c.JSON(http.StatusOK, response.Ok(user))
}

func (h *Handler) RefreshToken(c *echo.Context) error {
	var body struct {
		RefreshToken string `json:"refresh_token" validate:"required"`
	}

	if err := c.Bind(&body); err != nil {
		return c.JSON(http.StatusBadRequest, response.BadRequest(err.Error()))
	}

	if err := c.Validate(&body); err != nil {
		return c.JSON(http.StatusBadRequest, response.BadRequest(err.Error()))
	}

	newTokens, err := h.authService.Refresh(body.RefreshToken)
	if err != nil {
		return c.JSON(http.StatusBadRequest, response.BadRequest(err.Error()))
	}

	return c.JSON(http.StatusOK, response.Ok(newTokens))
}
