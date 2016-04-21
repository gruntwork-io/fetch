package main

import (
	"sort"

	"github.com/hashicorp/go-version"
	"strings"
)

func getLatestAcceptableTag(tagConstraint string, tags []string) (string, *fetchError) {

	// Sort all tags
	// Our use of the library go-version means that each tag will each be represented as a *version.Version
	versions := make([]*version.Version, len(tags))
	for i, tag := range tags {
		v, err := version.NewVersion(tag)
		if err != nil {
			return "", wrapError(err)
		}

		versions[i] = v
	}
	sort.Sort(version.Collection(versions))

	// If the tag constraint is empty, just return the latest
	if tagConstraint == "" {
		return versions[len(versions)-1].String(), nil
	}

	// Find the latest version that matches the given tag constraint
	constraints, err := version.NewConstraint(tagConstraint)
	if err != nil {
		// Explicitly check for a malformed tag value so we can return a nice error to the user
		if strings.Contains(err.Error(), "Malformed constraint") {
			return "", newError(100, err.Error())
		} else {
			return "", wrapError(err)
		}
	}

	latestAcceptableVersion := versions[0]
	for _, version := range versions {
		if constraints.Check(version) && version.GreaterThan(latestAcceptableVersion) {
			latestAcceptableVersion = version
		}
	}

	// The tag name may have started with a "v" or other string. If so, re-apply that string now
	var latestTag string
	for _, originalTagName := range tags {
		if strings.Contains(originalTagName, latestAcceptableVersion.String()) {
			latestTag = originalTagName
		}
	}

	return latestTag, nil
}
