package book

import "errors"

var ErrHoldNotAvailable = errors.New("book not available")
var ErrBookNotFound = errors.New("book not found")
