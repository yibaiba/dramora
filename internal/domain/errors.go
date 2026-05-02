package domain

import "errors"

var ErrInvalidTransition = errors.New("invalid status transition")
var ErrInvalidInput = errors.New("invalid input")
var ErrNotFound = errors.New("not found")
var ErrAlreadyDebited = errors.New("operation already debited")
