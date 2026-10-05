package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"ticketing/internal/app"
	"ticketing/internal/auth"
	"ticketing/internal/event"
	"ticketing/internal/order"
	"ticketing/internal/user"
	"ticketing/pkg/config"
	"ticketing/pkg/db"
	"ticketing/pkg/redis"
	"time"

	"github.com/labstack/echo/v5"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	cfg, err := config.New(".env")
	if err != nil {
		log.Fatalf(`fatal error config file "%s"`, err)
	}

	database, err := db.Connect(cfg)
	if err != nil {
		log.Fatal(`fatal error config file "%s"`, err)
	}

	if err := database.AutoMigrate(
		user.User{},
		auth.RefreshToken{},
		event.Event{},
		event.Seat{},
		order.Order{},
		order.OrderSeat{},
	); err != nil {
		log.Fatalf(`fatal error on DB auto migrate "%s"`, err)
	}

	rdb := redis.Connect(*cfg)
	defer rdb.Close()

	e := app.New(*cfg, database, rdb)

	sc := echo.StartConfig{
		Address:         ":8080",
		GracefulTimeout: 5 * time.Second,
	}

	if err := sc.Start(ctx, e); err != nil {
		log.Fatalf(`fatal echo start error: %s`, err)
	}
}
