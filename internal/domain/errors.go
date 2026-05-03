package domain

import "errors"

var ErrInvalidTransition = errors.New("invalid status transition")
var ErrInvalidInput = errors.New("invalid input")
var ErrNotFound = errors.New("not found")
var ErrAlreadyDebited = errors.New("operation already debited")
var ErrMaxRetriesExceeded = errors.New("max retries exceeded")
var ErrJobNotFailed = errors.New("job must be in failed status to retry")
var ErrQueuePaused = errors.New("queue is paused")
