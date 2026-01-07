package gitlab

import (
	"testing"

	"github.com/gruntwork-io/fetch/source"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseUrl(t *testing.T) {
	t.Parallel()

	config := source.Config{
		ApiVersion: "v4",
		Logger:     nil,
	}

	src := NewGitLabSource(config)

	cases := []struct {
		repoUrl string
		owner   string
		name    string
		token   string
	}{
		{"https://gitlab.com/owner/project", "owner", "project", ""},
		{"http://gitlab.com/owner/project", "owner", "project", ""},
		{"https://www.gitlab.com/owner/project", "owner", "project", ""},
		{"https://gitlab.com/owner/project/", "owner", "project", ""},
		{"https://gitlab.com/owner/project.git", "owner", "project", ""},
		{"https://gitlab.com/owner/project?foo=bar", "owner", "project", "token"},
	}

	for _, tc := range cases {
		t.Run(tc.repoUrl, func(t *testing.T) {
			repo, err := src.ParseUrl(tc.repoUrl, tc.token)
			require.NoError(t, err)

			assert.Equal(t, tc.owner, repo.Owner)
			assert.Equal(t, tc.name, repo.Name)
			assert.Equal(t, tc.repoUrl, repo.Url)
			assert.Equal(t, tc.token, repo.Token)
		})
	}
}

func TestParseUrlNestedGroups(t *testing.T) {
	t.Parallel()

	config := source.Config{
		ApiVersion: "v4",
		Logger:     nil,
	}

	src := NewGitLabSource(config)

	// GitLab supports nested groups (subgroups)
	cases := []struct {
		repoUrl string
		owner   string
		name    string
	}{
		{"https://gitlab.com/group/subgroup/project", "group/subgroup", "project"},
		{"https://gitlab.com/a/b/c/project", "a/b/c", "project"},
		{"https://gitlab.com/org/team/service/api", "org/team/service", "api"},
	}

	for _, tc := range cases {
		t.Run(tc.repoUrl, func(t *testing.T) {
			repo, err := src.ParseUrl(tc.repoUrl, "")
			require.NoError(t, err)

			assert.Equal(t, tc.owner, repo.Owner, "owner/namespace mismatch")
			assert.Equal(t, tc.name, repo.Name, "project name mismatch")
		})
	}
}

func TestParseUrlMalformed(t *testing.T) {
	t.Parallel()

	config := source.Config{
		ApiVersion: "v4",
		Logger:     nil,
	}

	src := NewGitLabSource(config)

	cases := []struct {
		repoUrl     string
		description string
	}{
		{"https://gitlab.com/project", "Missing owner (only one path segment)"},
	}

	for _, tc := range cases {
		t.Run(tc.description, func(t *testing.T) {
			_, err := src.ParseUrl(tc.repoUrl, "")
			assert.Error(t, err)
		})
	}
}

func TestParseUrlSelfHosted(t *testing.T) {
	t.Parallel()

	config := source.Config{
		ApiVersion: "v4",
		Logger:     nil,
	}

	src := NewGitLabSource(config)

	// Self-hosted GitLab instances
	cases := []struct {
		repoUrl string
		owner   string
		name    string
		apiUrl  string
	}{
		{"https://git.company.com/team/project", "team", "project", "git.company.com"},
		{"https://gitlab.internal.io/org/repo", "org", "repo", "gitlab.internal.io"},
	}

	for _, tc := range cases {
		t.Run(tc.repoUrl, func(t *testing.T) {
			repo, err := src.ParseUrl(tc.repoUrl, "")
			require.NoError(t, err)

			assert.Equal(t, tc.owner, repo.Owner)
			assert.Equal(t, tc.name, repo.Name)
			assert.Equal(t, tc.apiUrl, repo.ApiUrl)
		})
	}
}

func TestType(t *testing.T) {
	config := source.Config{
		ApiVersion: "v4",
		Logger:     nil,
	}

	src := NewGitLabSource(config)
	assert.Equal(t, source.TypeGitLab, src.Type())
}

func TestEncodeProjectPath(t *testing.T) {
	t.Parallel()

	cases := []struct {
		owner    string
		name     string
		expected string
	}{
		{"owner", "project", "owner%2Fproject"},
		{"group/subgroup", "project", "group%2Fsubgroup%2Fproject"},
		{"a/b/c", "repo", "a%2Fb%2Fc%2Frepo"},
	}

	for _, tc := range cases {
		t.Run(tc.owner+"/"+tc.name, func(t *testing.T) {
			encoded := encodeProjectPath(tc.owner, tc.name)
			assert.Equal(t, tc.expected, encoded)
		})
	}
}
