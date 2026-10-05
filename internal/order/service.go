package order

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"ticketing/internal/book"
	"ticketing/internal/event"

	"github.com/redis/go-redis/v9"
)

type service struct {
	rdb          *redis.Client
	repo         Repo
	eventService event.Service
}

type Service interface {
	CreateOrder(ctx context.Context, createOrder *CreateOrder, userID uint) (*Order, error)
}

func NewService(rdb *redis.Client, repo Repo, eventService event.Service) Service {
	return &service{rdb, repo, eventService}
}

func (s *service) CreateOrder(ctx context.Context, createOrder *CreateOrder, userID uint) (*Order, error) {
	data, err := s.rdb.Get(ctx, "hold:"+createOrder.HoldID).Bytes()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, book.ErrBookNotFound
		}
		return nil, err
	}

	var hold book.Hold
	if err := json.Unmarshal(data, &hold); err != nil {
		return nil, err
	}

	if hold.UserID != userID {
		return nil, book.ErrBookNotFound
	}

	var orderSeats []OrderSeat

	for _, seatID := range hold.SeatIDs {
		orderSeats = append(orderSeats, OrderSeat{
			SeatID: seatID,
		})
	}

	order := &Order{
		EventID: hold.EventID,
		UserID:  hold.UserID,
		Seats:   orderSeats,
	}

	if err := s.repo.CreateOrder(ctx, order); err != nil {
		return nil, err
	}

	for _, seatID := range hold.SeatIDs {
		if err := s.eventService.UpdateSeatStatus(ctx, seatID, "booked"); err != nil {
			return nil, err
		}

		if _, err := s.rdb.Del(ctx, fmt.Sprintf("seat:%d:%d", seatID, hold.EventID)).Result(); err != nil {
			return nil, err
		}
	}

	if _, err := s.rdb.Del(ctx, "hold:"+createOrder.HoldID).Result(); err != nil {
		return nil, err
	}

	return order, nil
}
