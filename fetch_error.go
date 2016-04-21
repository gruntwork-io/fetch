package main

import "fmt"

// We define a custom error type so that we can provide friendlier error messages
type fetchError struct {
	errorCode int    // an error code is an arbitrary int that allows for strongly typed identification of specific errors
	details   string // the output of the underlying error message, if any
	err       error  // the underlying golang error, if any
}

// Implement the golang Error interface
func (e *fetchError) Error() string {
	return fmt.Sprintf("%d - %s", e.errorCode, e.details)
}

func newError(errorCode int, details string) *fetchError {
	return &fetchError{
		errorCode: errorCode,
		details: details,
		err: nil,
	}
}

func wrapError(err error) *fetchError {
	return &fetchError{
		errorCode: -1,
		details: err.Error(),
		err: err,
	}
}
