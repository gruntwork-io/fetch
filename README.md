# fetch

fetch makes it easy to download files, folders, and release assets from a specific git commit, branch, or tag of
public and private GitHub repos.

#### Features

- Download from a specific git commit SHA.
- Download from a specific git tag.
- Download from a specific git branch.
- Download a single source file, a subset of source files, or all source files from the repo.
- Download a binary asset from a specific release.
- Download from public repos.
- Download from private repos by specifying a [GitHub Personal Access Token](https://help.github.com/articles/creating-an-access-token-for-command-line-use/).
- When specifying a git tag, you can can specify either exactly the tag you want, or a [Tag Constraint Expression](#tag-constraint-expressions) to do things like  "get the latest non-breaking version" of this repo. Note that fetch assumes git tags are specified according to [Semantic Versioning](http://semver.org/) principles.

## Motivation

[Gruntwork](http://gruntwork.io) helps software teams get up and running on AWS with DevOps best practices and world-class 
infrastructure in about 2 weeks. Sometimes we publish scripts and binaries that clients use in their infrastructure,
and we want clients to auto-download the latest non-breaking version of that script or binary when we publish updates.
In addition, for security reasons, we wish to verify the integrity of the git commit being downloaded.

## Installation

Download the fetch binary from the [GitHub Releases](https://github.com/gruntwork-io/fetch/releases) tab.

## Assumptions

fetch assumes that a repo's tags are in the format `vX.Y.Z` or `X.Y.Z` to support Semantic Versioning parsing. Repos
that use git tags not in this format cannot currently be used with fetch.

## Usage

#### General Usage

```
fetch [OPTIONS] <local-download-path>
```

The supported options are:

- `--repo` (**Required**): The fully qualified URL of the GitHub repo to download from (e.g. https://github.com/foo/bar).
- `--tag` (**Optional**): The git tag to download. Can be a specific tag or a [Tag Constraint
  Expression](#tag-constraint-expressions).
- `--branch` (**Optional**): The git branch from which to download; the latest commit in the branch will be used. If
  specified, will override `--tag`.
- `--commit` (**Optional**): The SHA of a git commit to download. If specified, will override `--branch` and `--tag`.
- `--source-path` (**Optional**): The source path to download from the repo (e.g. `--source-path=/folder` will download
  the `/folder` path and all files below it). By default, all files are downloaded from the repo unless `--source-path`
  or `--release-asset` is specified. This option can be specified more than once.
- `--release-asset` (**Optional**): The name of a release asset--that is, a binary uploaded to a [GitHub
  Release](https://help.github.com/articles/creating-releases/)--to download. This option can be specified more than
  once. It only works with the `--tag` option.
- `--github-oauth-token` (**Optional**): A [GitHub Personal Access
  Token](https://help.github.com/articles/creating-an-access-token-for-command-line-use/). Required if you're
  downloading from private GitHub repos. **NOTE:** fetch will also look for this token using the `GITHUB_OAUTH_TOKEN`
  environment variable, which we recommend using instead of the command line option to ensure the token doesn't get
  saved in bash history.

The supported arguments are:

- `<local-download-path>` (**Required**): The local path where all files should be downloaded (e.g. `/tmp`).

Run `fetch --help` to see more information about the flags.

#### Usage Example 1

Download `/modules/foo/bar.sh` from a GitHub release where the tag is the latest version of `0.1.x` but at least `0.1.5`, and save it to `/tmp/bar`:

```
fetch \
--repo="https://github.com/gruntwork-io/script-modules" \
--tag="~>0.1.5" \
--source-path="/modules/foo/bar.sh" \
/tmp/bar
```

#### Usage Example 2

Download all files in `/modules/foo` from a GitHub release where the tag is exactly `0.1.5`, and save them to `/tmp`:

```
fetch \
--repo="https://github.com/gruntwork-io/script-modules" \
--tag="0.1.5" \
--source-path="/modules/foo" \
/tmp

```

#### Usage Example 3

Download all files from a private GitHub repo using the GitHUb oAuth Token `123`. Get the release whose tag is exactly `0.1.5`, and save the files to `/tmp`:

```
GITHUB_OAUTH_TOKEN=123

fetch \
--repo="https://github.com/gruntwork-io/script-modules" \
--tag="0.1.5" \
/tmp

```

#### Usage Example 4

Download all files from the latest commit on the `sample-branch` branch, and save them to `/tmp`:

```
fetch \
--repo="https://github.com/gruntwork-io/fetch-test-public" \
--branch="sample-branch" \
/tmp/josh1

```

#### Usage Example 5

Download all files from the git commit `f32a08313e30f116a1f5617b8b68c11f1c1dbb61`, and save them to `/tmp`:

```
fetch \
--repo="https://github.com/gruntwork-io/fetch-test-public" \
--commit="f32a08313e30f116a1f5617b8b68c11f1c1dbb61" \
/tmp/josh1

```

#### Usage Example 6

Download the release asset `foo.exe` from a GitHub release where the tag is exactly `0.1.5`, and save it to `/tmp`:

```
fetch \
--repo="https://github.com/gruntwork-io/script-modules" \
--tag="0.1.5" \
--release-asset="foo.exe" \
/tmp
```

#### Tag Constraint Expressions

The value of `--tag` can be expressed using any operators defined in [hashicorp/go-version](https://github.com/hashicorp/go-version).

Specifically, this includes:

| Tag Constraint Pattern | Meaning                                  |
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

- Introduce code verification using something like GPG signatures or published checksums
- Explicitly test for exotic repo and org names
- Apply stricter parsing for repo-filter command-line arg
