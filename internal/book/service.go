package book

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"ticketing/internal/event"
	"time"
	"uuid"

	"github.com/redis/go-redis/v9"
)

type service struct {
	rdb           *redis.Client
	eventsService event.Service
}

type Service interface {
	Book(ctx context.Context, createHold *CreateHold, userID uint) (*Hold, error)
}

func NewService(rdb *redis.Client, eventsService event.Service) Service {
	return &service{rdb, eventsService}
}

func (s *service) Book(ctx context.Context, createHold *CreateHold, userID uint) (*Hold, error) {
	event, err := s.eventsService.GetEventByID(ctx, createHold.EventID)
	if err != nil {
		return nil, err
	}

	var seats = make(map[uint]bool)

	for _, seatID := range createHold.SeatIDs {
		seats[seatID] = true
	}

	for _, seat := range event.Seats {
		if slices.Contains(createHold.SeatIDs, seat.ID) {
			if seat.Status != "available" {
				return nil, ErrHoldNotAvailable
			}

			delete(seats, seat.ID)
		}
	}

	if len(seats) > 0 {
		return nil, errors.New(fmt.Sprintf(`seats not available: %v`, seats))
	}

	hold := &Hold{
		ID:        uuid.New(),
		EventID:   createHold.EventID,
		SeatIDs:   createHold.SeatIDs,
		UserID:    userID,
		ExpiresAt: time.Now().Add(time.Minute * 10),
	}
	data, err := json.Marshal(hold)
	if err != nil {
		return nil, err
	}

	if _, err := s.rdb.SetNX(ctx, "hold:"+hold.ID.String(), data, time.Minute*10).Result(); err != nil {
		return nil, err
	}
	for _, seatID := range hold.SeatIDs {
		ok, err := s.rdb.SetNX(ctx, fmt.Sprintf(`seat:%d:%d`, seatID, createHold.EventID), hold.ID.String(), time.Minute*10).Result()
		if err != nil {
			return nil, err
		}
		if !ok {
			return nil, ErrHoldNotAvailable
		}
	}

	return hold, nil
}
