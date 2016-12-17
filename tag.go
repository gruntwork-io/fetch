package main

import (
	"sort"
	"strings"

	"github.com/hashicorp/go-version"
)

func isTagConstraintSpecificTag(tagConstraint string) (bool, string) {
	if len(tagConstraint) > 0 {
		switch tagConstraint[0] {
		// Check for a tagConstraint '='
		case '=':
			return true, strings.TrimSpace(tagConstraint[1:])

		// Check for a tagConstraint without constraint specifier
		// Neither of '!=', '>', '>=', '<', '<=', '~>' is prefixed before tag
		case '>', '<', '!', '~':
			return false, tagConstraint

		default:
			return true, strings.TrimSpace(tagConstraint)
		}
	}
	return false, tagConstraint
}

func getLatestAcceptableTag(tagConstraint string, tags []string) (string, *FetchError) {
	var latestTag string

	if len(tags) == 0 {
		return latestTag, nil
	}

	// Sort all tags
	// Our use of the library go-version means that each tag will each be represented as a *version.Version
	versions := make([]*version.Version, len(tags))
	for i, tag := range tags {
		v, err := version.NewVersion(tag)
		if err != nil {
			return latestTag, wrapError(err)
		}

		versions[i] = v
	}
	sort.Sort(version.Collection(versions))

	// If the tag constraint is empty, set it to the latest tag
	if tagConstraint == "" {
		tagConstraint = versions[len(versions)-1].String()
	}

	// Find the latest version that matches the given tag constraint
	constraints, err := version.NewConstraint(tagConstraint)
	if err != nil {
		// Explicitly check for a malformed tag value so we can return a nice error to the user
		if strings.Contains(err.Error(), "Malformed constraint") {
			return latestTag, newError(INVALID_TAG_CONSTRAINT_EXPRESSION, err.Error())
		} else {
			return latestTag, wrapError(err)
		}
	}

	latestAcceptableVersion := versions[0]
	for _, version := range versions {
		if constraints.Check(version) && version.GreaterThan(latestAcceptableVersion) {
			latestAcceptableVersion = version
		}
	}

	// The tag name may have started with a "v". If so, re-apply that string now
	for _, originalTagName := range tags {
		if strings.Contains(originalTagName, latestAcceptableVersion.String()) {
			latestTag = originalTagName
		}
	}

	return latestTag, nil
}
