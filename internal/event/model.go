package event

import (
	"time"

	"gorm.io/gorm"
)

type Event struct {
	gorm.Model
	Name      string    `json:"name" validate:"required"`
	Capacity  string    `json:"capacity" validate:"required,num_pair"`
	Price     uint      `json:"price" validate:"required,number"`
	StartTime time.Time `json:"start_time" validate:"required"`
	Seats     []Seat    `json:"seats"`
}

type CreateEvent struct {
	Name      string    `json:"name" validate:"required"`
	StartTime time.Time `json:"start_time" validate:"required"`
	Capacity  string    `json:"capacity" validate:"required,num_pair"`
	Price     uint      `json:"price" validate:"required,number"`
}

type Seat struct {
	gorm.Model
	Status   string `json:"status"`
	EventID  uint   `gorm:"index"`
	Location string `json:"location" validate:"required,num_pair"`
}
