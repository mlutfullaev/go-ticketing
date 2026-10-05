package app

import (
	"log/slog"
	"net/http"
	"os"
	"ticketing/internal/auth"
	"ticketing/internal/book"
	"ticketing/internal/event"
	"ticketing/internal/order"
	"ticketing/internal/user"
	"ticketing/pkg/config"
	"ticketing/pkg/log"
	"ticketing/pkg/validate"

	"github.com/labstack/echo/v5"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

func New(cfg config.Config, db *gorm.DB, rdb *redis.Client) *echo.Echo {
	e := echo.New()

	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	e.Use(echo.WrapMiddleware(func(next http.Handler) http.Handler {
		return log.LoggerMiddleware(logger, next)
	}))

	e.Validator = validate.New()

	authMiddleware := auth.Middleware(cfg.JWT.Secret)

	userRepo := user.NewUserRepository(db)
	userService := user.NewUserService(userRepo)
	authRepo := auth.NewRepository(db)
	authService := auth.NewAuthService(userService, cfg.JWT, authRepo)
	authHandler := auth.NewHandler(authService)

	authHandler.RegisterRoutes(e.Group("/auth"))

	eventRepo := event.NewRepository(db)
	eventService := event.NewService(eventRepo, rdb)
	eventHandler := event.NewHandler(eventService)

	eventHandler.RegisterRoutes(e.Group("/events"), authMiddleware)
	bookService := book.NewService(rdb, eventService)
	bookHandler := book.NewHandler(bookService)

	bookHandler.RegisterRoutes(e.Group("/book"), authMiddleware)

	orderRepo := order.NewRepository(db)
	orderService := order.NewService(rdb, orderRepo, eventService)
	orderHandler := order.NewHandler(orderService)

	orderHandler.RegisterRoutes(e.Group("/order"), authMiddleware)

	return e
}
