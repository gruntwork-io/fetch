package source

import (
	"fmt"
	"net/url"
	"strings"
)

// DetectSourceType determines provider from URL
// Only auto-detects public github.com and gitlab.com
// For custom/self-hosted domains, user must specify --source flag
// Returns GitHub as default for unknown hosts (backward compatibility)
func DetectSourceType(repoUrl string) (SourceType, error) {
	u, err := url.Parse(repoUrl)
	if err != nil {
		return "", fmt.Errorf("invalid URL: %w", err)
	}

	host := strings.ToLower(u.Host)

	// Exact match for public GitLab only
	if host == "gitlab.com" || host == "www.gitlab.com" {
		return SourceTypeGitLab, nil
	}

	// Exact match for public GitHub
	if host == "github.com" || host == "www.github.com" {
		return SourceTypeGitHub, nil
	}

	// Default to GitHub for unknown domains (backward compatibility)
	// Users with custom GitLab domains should use --source gitlab
	return SourceTypeGitHub, nil
}

// ParseSourceType converts string to SourceType
func ParseSourceType(s string) (SourceType, error) {
	switch strings.ToLower(s) {
	case "github":
		return SourceTypeGitHub, nil
	case "gitlab":
		return SourceTypeGitLab, nil
	case "auto", "":
		return SourceTypeAuto, nil
	default:
		return "", fmt.Errorf("unknown source type: %s (valid: github, gitlab, auto)", s)
	}
}

// GetSource auto-detects or uses explicit type to create a Source implementation
func GetSource(repoUrl string, explicitType SourceType, config Config) (Source, error) {
	var srcType SourceType
	var err error

	if explicitType != "" && explicitType != SourceTypeAuto {
		srcType = explicitType
	} else {
		srcType, err = DetectSourceType(repoUrl)
		if err != nil {
			return nil, err
		}
	}

	return NewSource(srcType, config)
}

// NewSource creates a Source implementation based on type
func NewSource(sourceType SourceType, config Config) (Source, error) {
	switch sourceType {
	case SourceTypeGitHub:
		return NewGitHubSource(config), nil
	case SourceTypeGitLab:
		return NewGitLabSource(config), nil
	default:
		return nil, fmt.Errorf("unsupported source type: %s", sourceType)
	}
}

// NewGitHubSource creates a GitHub source - placeholder until github package is created
var NewGitHubSource func(config Config) Source

// NewGitLabSource creates a GitLab source - placeholder until gitlab package is created
var NewGitLabSource func(config Config) Source
