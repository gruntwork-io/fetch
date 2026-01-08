package source

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDetectSourceType(t *testing.T) {
	t.Parallel()

	cases := []struct {
		repoUrl      string
		expectedType SourceType
		description  string
	}{
		// GitHub URLs
		{"https://github.com/owner/repo", TypeGitHub, "github.com"},
		{"https://www.github.com/owner/repo", TypeGitHub, "www.github.com"},
		{"http://github.com/owner/repo", TypeGitHub, "http github.com"},

		// GitLab URLs
		{"https://gitlab.com/owner/repo", TypeGitLab, "gitlab.com"},
		{"https://www.gitlab.com/owner/repo", TypeGitLab, "www.gitlab.com"},
		{"http://gitlab.com/owner/repo", TypeGitLab, "http gitlab.com"},

		// Custom domains default to GitHub (backward compatibility)
		{"https://git.company.com/owner/repo", TypeGitHub, "custom domain defaults to GitHub"},
		{"https://ghe.mycompany.com/owner/repo", TypeGitHub, "GitHub Enterprise domain"},
		{"https://gitlab.internal.com/owner/repo", TypeGitHub, "custom GitLab domain defaults to GitHub"},
	}

	for _, tc := range cases {
		t.Run(tc.description, func(t *testing.T) {
			detected, err := DetectSourceType(tc.repoUrl)
			require.NoError(t, err)
			assert.Equal(t, tc.expectedType, detected)
		})
	}
}

func TestDetectSourceTypeInvalidURL(t *testing.T) {
	t.Parallel()

	_, err := DetectSourceType("://invalid-url")
	assert.Error(t, err)
}

func TestParseSourceType(t *testing.T) {
	t.Parallel()

	cases := []struct {
		input        string
		expectedType SourceType
		description  string
	}{
		{"github", TypeGitHub, "lowercase github"},
		{"GitHub", TypeGitHub, "mixed case GitHub"},
		{"GITHUB", TypeGitHub, "uppercase GITHUB"},
		{"gitlab", TypeGitLab, "lowercase gitlab"},
		{"GitLab", TypeGitLab, "mixed case GitLab"},
		{"GITLAB", TypeGitLab, "uppercase GITLAB"},
		{"auto", TypeAuto, "lowercase auto"},
		{"Auto", TypeAuto, "mixed case Auto"},
		{"", TypeAuto, "empty string defaults to auto"},
	}

	for _, tc := range cases {
		t.Run(tc.description, func(t *testing.T) {
			parsed, err := ParseSourceType(tc.input)
			require.NoError(t, err)
			assert.Equal(t, tc.expectedType, parsed)
		})
	}
}

func TestParseSourceTypeInvalid(t *testing.T) {
	t.Parallel()

	invalidTypes := []string{"unknown", "bitbucket", "svn", "hg"}

	for _, invalid := range invalidTypes {
		t.Run(invalid, func(t *testing.T) {
			_, err := ParseSourceType(invalid)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "unknown source type")
		})
	}
}

func TestNewSourceUnsupportedType(t *testing.T) {
	t.Parallel()

	config := Config{
		ApiVersion: "v3",
		Logger:     nil,
	}

	_, err := NewSource("unsupported", config)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported source type")
}
