package auth

import "errors"

var ErrTokenNotFound = errors.New("token not found")
var ErrTokenExpired = errors.New("token expired")
