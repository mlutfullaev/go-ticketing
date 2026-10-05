package event

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/redis/go-redis/v9"
)

type service struct {
	repository Repository
	rdb        *redis.Client
}

type Service interface {
	CreateEvent(createEvent *CreateEvent) (*Event, error)
	GetEvents() ([]*Event, error)
	GetEventByID(ctx context.Context, eventID uint) (*Event, error)
	UpdateSeatStatus(ctx context.Context, seatID uint, status string) error
}

func NewService(repository Repository, rdb *redis.Client) Service {
	return &service{repository, rdb}
}

func (s *service) CreateEvent(createEvent *CreateEvent) (*Event, error) {
	event := &Event{
		Name:      createEvent.Name,
		StartTime: createEvent.StartTime,
		Capacity:  createEvent.Capacity,
		Price:     createEvent.Price,
	}

	row, col, err := getCapacity(event.Capacity)
	if err != nil {
		return nil, err
	}

	for i := range row {
		for j := range col {
			seat := Seat{
				Location: fmt.Sprintf(`%d:%d`, i, j),
				EventID:  event.ID,
				Status:   "available",
			}

			event.Seats = append(event.Seats, seat)
		}
	}

	if err := s.repository.CreateEvent(event); err != nil {
		return nil, err
	}

	return event, nil
}

func (s *service) GetEvents() ([]*Event, error) {
	return s.repository.GetEvents()
}

func (s *service) GetEventByID(ctx context.Context, eventID uint) (*Event, error) {
	event, err := s.repository.GetEventByID(eventID)
	if err != nil {
		return nil, err
	}

	seatIds := make([]string, len(event.Seats))
	for i, seat := range event.Seats {
		seatIds[i] = fmt.Sprintf(`seat:%d:%d`, seat.ID, eventID)
	}

	values, err := s.rdb.MGet(ctx, seatIds...).Result()
	if err != nil {
		return nil, err
	}
	for i, v := range values {
		if v != nil && event.Seats[i].Status == "available" {
			event.Seats[i].Status = "held"
		}
	}

	return event, nil
}

func (s *service) UpdateSeatStatus(ctx context.Context, seatID uint, status string) error {
	return s.repository.UpdateSeatStatus(ctx, seatID, status)
}

const MaxCapacity = 1_000

func getCapacity(capacity string) (uint, uint, error) {
	separate := strings.Split(capacity, ":")
	if len(separate) != 2 {
		return 0, 0, ErrInvalidCapacity
	}

	row, rowErr := strconv.ParseUint(separate[0], 10, 64)
	col, colErr := strconv.ParseUint(separate[1], 10, 64)

	if rowErr != nil || colErr != nil {
		return 0, 0, ErrInvalidCapacity
	}

	if row < 1 || col < 1 {
		return 0, 0, ErrInvalidCapacity
	}

	if row > MaxCapacity || col > MaxCapacity {
		return 0, 0, ErrMaxCapacityExceeded
	}

	return uint(row), uint(col), nil
}
