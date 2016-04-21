# fetch

fetch will download a file or folder from a specific tag of a GitHub repo, subject to the [Semantic Versioning](http://semver.org/) 
constraints you impose. It works for public repos, and for private repos by allowing you to specify a GitHub [Personal Access Token](https://help.github.com/articles/creating-an-access-token-for-command-line-use/).

It is well-suited to downloading the latest version of a file or folder published in a GitHub repo such that you get 
the latest non-breaking-change version. Basically, it's like a package manager, but for arbitrary GitHub repos.

## Motivation
[Gruntwork](http://gruntwork.io) helps software teams get up and running on AWS with DevOps best practices and world-class 
infrastructure in about 2 weeks. Sometimes we publish scripts that clients use in their infrastructure, and we want clients
to auto-download the latest non-breaking version of a script when we publish updates. In addition, for security reasons,
we wish to verify the integrity of the git commit being downloaded.
 
## Installation
Download the binary from the [GitHub Releases](https://github.com/gruntwork-io/script-modules/releases) tab. 

## Assumptions
fetch assumes that a repo's tags are in the format `vX.Y.Z` or `X.Y.Z` to support Semantic Versioning parsing. Repos that
use git tags not in this format cannot be used with fetch.

## Usage

#### General Usage
```
fetch \
--repo=<github-repo-url> --tag=<version-constraint> /repo/path/to/file/or/directory /output/path/to/file/or/directory
```

Run `fetch --help` to see more information about each argument. See [Version Constraint Operators](#version-constraint-operators)
for examples of version constraints you can use.

#### Example 1

Download `/modules/foo/bar.sh` from a GitHub tagged release where the tag is the latest version of 0.1.x but at least 0.1.5, and save it to `/tmp/bar`:

```
fetch \
--repo="https://github.com/gruntwork-io/script-modules" \
--tag="~>0.1.5" \
/modules/foo/bar.sh \
/tmp/bar
```

#### Example 2

Download all files in `/modules/foo` from a GitHub tagged release where the tag is exactly 0.1.5, and save them to `/tmp`:

```
fetch \
--repo="https://github.com/gruntwork-io/script-modules" \
--tag="0.1.5" \
/modules/foo \
/tmp

```

#### Example 3

Download all files in `/modules/foo` from a private GitHub repo using the GitHUb oAuth Token "123". Get the release whose tag is exactly 0.1.5, and save the files to `/tmp`:

```
fetch \
--repo="https://github.com/gruntwork-io/script-modules" \
--tag="0.1.5" \
--github-oauth-token="123" \
/modules/foo \
/tmp

```

#### Version Constraint Operators

Version contraints can be expressed using any operators defined in [hashicorp/go-version](https://github.com/hashicorp/go-version).

Specifically, this includes:

| Version Constraint Pattern | Meaning                                  |
| -------------------------- | ---------------------------------------- |
| `1.0.7`                    | Exactly version `1.0.7`                  |
| `=1.0.7`                   | Exactly version `1.0.7`                  |
| `!=1.0.7`                  | The latest version as long as that version is not `1.0.7` |
| `>1.0.7`                   | The latest version greater than `1.0.7`  |
| `<1.0.7`                   | The latest version that's less than `1.0.7` |
| `>=1.0.7`                  | The latest version greater than or equal to `1.0.7` |
| `<=1.0.7`                  | The latest version that's less than or equal to `1.0.7` |
| `~>1.0.7`                  | The latest version that is greater than `1.0.7` and less than `1.1.0` |
| `~>1.0`                    | The latest version that is greater than `1.0` and less than `2.0` |

## TODO
- Capture final two args properly from CLI
- Add circle.yml
- Introduce code verification using something like GPG signatures or published checksums
- Explicitly test for exotic repo and org names
- Consider streamlining code for error handling