package main

import (
	"testing"
)

func TestGetLatestAcceptableTag(t *testing.T) {
	t.Parallel()

	cases := []struct {
		tagConstraint string
		tags          []string
		expectedTag   string
	}{
		{"1.0.7", []string{"1.0.7"}, "1.0.7"},

		{"~> 1.0.0", []string{"1.0.5","1.0.6","1.0.7","1.0.8","1.0.9","1.1.0","1.2.3"}, "1.0.9"},
		{"~> 1.0.7", []string{"1.0.5","1.0.6","1.0.7","1.0.8","1.0.9","1.1.0","1.2.3"}, "1.0.9"},
		{"~> 1.1.0", []string{"1.0.5","1.0.6","1.0.7","1.0.8","1.0.9","1.1.0","1.2.3"}, "1.1.0"},
		{"~> 1.1.1", []string{"1.0.5","1.0.6","1.0.7","1.0.8","1.0.9","1.1.0","1.1.1","1.1.2","1.1.3","1.2.3","1.4.0","2.0.0","2.1.0"}, "1.1.3"},
		{"~> 1.2.1", []string{"1.0.5","1.0.6","1.0.7","1.0.8","1.0.9","1.1.0","1.1.1","1.1.2","1.1.3","1.2.3","1.4.0","2.0.0","2.1.0"}, "1.2.3"},
		{"~> 1.1", []string{"1.0.5","1.0.6","1.0.7","1.0.8","1.0.9","1.1.0","1.1.1","1.1.2","1.1.3","1.2.3","1.4.0","2.0.0","2.1.0"}, "1.4.0"},
		{"~> 1.2", []string{"1.0.5","1.0.6","1.0.7","1.0.8","1.0.9","1.1.0","1.1.1","1.1.2","1.1.3","1.2.3","1.4.0","2.0.0","2.1.0"}, "1.4.0"},
		{"~> 1.3", []string{"1.0.5","1.0.6","1.0.7","1.0.8","1.0.9","1.1.0","1.1.1","1.1.2","1.1.3","1.2.3","1.4.0","2.0.0","2.1.0"}, "1.4.0"},

		{">= 1.3", []string{"1.0.5","1.0.6","1.0.7","1.0.8","1.0.9","1.1.0","1.1.1","1.1.2","1.1.3","1.2.3","1.4.0","2.0.0","2.1.0"}, "2.1.0"},

		{"v1.0.7", []string{"v1.0.7"}, "v1.0.7"},
		{"v1.0.7", []string{}, ""},
	}

	for _, tc := range cases {
		tag, err := getLatestAcceptableTag(tc.tagConstraint, tc.tags)
		if err != nil {
			t.Fatalf("Failed on call to getLatestAcceptableTag: %s", err.details)
		}

		if tag != tc.expectedTag {
			t.Fatalf("Given constraint %s and tag list %v, expected %s, but received: %s", tc.tagConstraint, tc.tags, tc.expectedTag, tag)
		}
	}
}

func TestGetLatestAcceptableTagOnEmptyConstraint(t *testing.T) {
	t.Parallel()

	cases := []struct {
		tagConstraint string
		tags          []string
		expectedTag   string
	}{
		{"", []string{"v0.0.1","v0.0.2","v0.0.3"}, "v0.0.3"},
		{"", []string{"1.0.5","1.0.6","1.0.7","1.0.8","1.0.9","1.1.0","1.2.3"}, "1.2.3"},
		{"", []string{"1.0.5","1.0.6","1.0.7","1.0.8","1.0.9","1.1.0","1.1.1","1.1.2","1.1.3","1.2.3","1.4.0","2.0.0","2.1.0"}, "2.1.0"},
		{"", []string{}, ""},
	}

	for _, tc := range cases {
		tag, err := getLatestAcceptableTag(tc.tagConstraint, tc.tags)
		if err != nil {
			t.Fatalf("Failed on call to getLatestAcceptableTag: %s", err.details)
		}

		if tag != tc.expectedTag {
			t.Fatalf("Given constraint %s and tag list %v, expected %s, but received: %s", tc.tagConstraint, tc.tags, tc.expectedTag, tag)
		}
	}
}

func TestGetLatestAcceptableTagOnMalformedConstraint(t *testing.T) {
	t.Parallel()

	cases := []struct {
		tagConstraint string
	}{
		{"josh"},
		{"plump elephants dancing in the night"},
	}

	for _, tc := range cases {
		_, err := getLatestAcceptableTag(tc.tagConstraint, []string{"v0.0.1"})
		if err == nil {
			t.Fatalf("Expected malformed constraint error, but received nothing.")
		}
	}
}
