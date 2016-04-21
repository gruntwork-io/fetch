package main

import (
	"testing"
)

func TestNewError(t *testing.T) {
	_ = newError(1, "My error details")
}