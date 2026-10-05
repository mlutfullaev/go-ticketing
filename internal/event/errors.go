package event

import (
	"errors"
	"fmt"
)

var ErrInvalidCapacity = errors.New("invalid capacity")
var ErrMaxCapacityExceeded = errors.New(fmt.Sprintf(`max capacity exceeded: %d`, MaxCapacity))
var ErrEventNotFound = errors.New("event not found")
var ErrSeatsAreNotAvailable = errors.New("seats are not available")
