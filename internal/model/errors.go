package model

import "errors"

var (
	ErrNotFound  = errors.New("not found")
	ErrForbidden = errors.New("forbidden")
	ErrConflict  = errors.New("already submitted")

	ErrUnauthorized    = errors.New("authentication required")
	ErrFormNotActive   = errors.New("form is not accepting responses")
	ErrFormNotOpenYet  = errors.New("form has not opened yet")
	ErrFormExpired     = errors.New("form has closed")
	ErrFormFull        = errors.New("form response limit reached")
)
