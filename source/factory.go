package source

import (
	"fmt"
	"net/url"
	"strings"
)

// DetectSourceType determines provider from URL
// Auto-detects GitHub or GitLab if the hostname contains "github" or "gitlab"
// Returns GitHub as default for unknown hosts (backward compatibility)
func DetectSourceType(repoUrl string) (SourceType, error) {
	u, err := url.Parse(repoUrl)
	if err != nil {
		return "", fmt.Errorf("invalid URL: %w", err)
	}

	host := strings.ToLower(u.Host)

	// Substring match for GitLab domains (e.g., gitlab.mycompany.com)
	if strings.Contains(host, "gitlab") {
		return TypeGitLab, nil
	}

	// Substring match for GitHub domains (e.g., github.mycompany.com)
	if strings.Contains(host, "github") {
		return TypeGitHub, nil
	}

	// Default to GitHub for unknown domains (backward compatibility)
	return TypeGitHub, nil
}

// ParseSourceType converts string to SourceType
func ParseSourceType(s string) (SourceType, error) {
	switch strings.ToLower(s) {
	case "github":
		return TypeGitHub, nil
	case "gitlab":
		return TypeGitLab, nil
	case "auto", "":
		return TypeAuto, nil
	default:
		return "", fmt.Errorf("unknown source type: %s (valid: github, gitlab, auto)", s)
	}
}

// GetSource auto-detects or uses explicit type to create a Source implementation
func GetSource(repoUrl string, explicitType SourceType, config Config) (Source, error) {
	var srcType SourceType
	var err error

	if explicitType != "" && explicitType != TypeAuto {
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
	case TypeGitHub:
		return NewGitHubSource(config), nil
	case TypeGitLab:
		return NewGitLabSource(config), nil
	default:
		return nil, fmt.Errorf("unsupported source type: %s", sourceType)
	}
}

// NewGitHubSource creates a GitHub source - placeholder until github package is created
var NewGitHubSource func(config Config) Source

// NewGitLabSource creates a GitLab source - placeholder until gitlab package is created
var NewGitLabSource func(config Config) Source
