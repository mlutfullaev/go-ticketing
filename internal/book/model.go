package book

import (
	"time"
	"uuid"
)

type Hold struct {
	UserID    uint      `redis:"user_id" json:"user_id"`
	EventID   uint      `redis:"event_id" json:"event_id"`
	SeatIDs   []uint    `redis:"seat_ids" json:"seat_ids"`
	ID        uuid.UUID `redis:"id" json:"id"`
	ExpiresAt time.Time `redis:"expires" json:"expires_at"`
}

type CreateHold struct {
	EventID uint   `json:"event_id" validate:"required"`
	SeatIDs []uint `json:"seat_ids" validate:"required"`
}
