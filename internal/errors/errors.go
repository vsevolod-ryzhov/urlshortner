// Package errors stores common application errors
package errors

import "errors"

var (
	ErrNotFound   = errors.New("record not found")
	ErrDeletedURL = errors.New("record deleted")
)
