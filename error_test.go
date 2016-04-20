package main

import (
	"testing"
)

func TestNewError(t *testing.T) {
	_ = newError(1, "My error details")
}

func TestErrorComparisonToNil(t *testing.T) {
	err := newEmptyError()
	if err != nil {
		t.Fatalf("Expected err to be nil")
	}
}