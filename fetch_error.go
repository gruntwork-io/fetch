package main

import "fmt"

// We define a custom error type so that we can provide friendlier error messages
type FetchError struct {
	errorCode int    // an error code is an arbitrary int that allows for strongly typed identification of specific errors
	details   string // the output of the underlying error message, if any
	err       error  // the underlying golang error, if any
}

// Implement the golang Error interface
func (e *FetchError) Error() string {
	return fmt.Sprintf("%d - %s", e.errorCode, e.details)
}

func newError(errorCode int, details string) *FetchError {
	return &FetchError{
		errorCode: errorCode,
		details:   details,
		err:       nil,
	}
}

func wrapError(err error) *FetchError {
	if err == nil {
		return nil
	}
	return &FetchError{
		errorCode: -1,
		details:   err.Error(),
		err:       err,
	}
}
