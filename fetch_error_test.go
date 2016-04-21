package main

import (
	"testing"
)

func TestNewError(t *testing.T) {
	t.Parallel()

	_ = newError(1, "My error details")
}