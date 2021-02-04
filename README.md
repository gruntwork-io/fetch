[![Maintained by Gruntwork.io](https://img.shields.io/badge/maintained%20by-gruntwork.io-%235849a6.svg)](https://gruntwork.io/?ref=repo_fetch)
# fetch

fetch makes it easy to download files, folders, or release assets from a specific commit, branch, or tag of
a public or private GitHub repo.

#### Motivation

[Gruntwork](http://gruntwork.io) helps software teams get up and running on AWS with DevOps best practices and
world-class infrastructure in about a day. Sometimes we publish scripts and binaries that clients use in their
infrastructure, and we want an easy way to install a specific version of one of those scripts and binaries. While this
is fairly straightforward to do with public GitHub repos, as you can usually `curl` or `wget` a public URL, it's much
trickier to do with private GitHub repos, as you have to make multiple API calls, parse JSON responses, and handle
authentication. Fetch makes it possible to handle all of these cases with a one-liner.

#### Features

- Download from any git reference, such as a specific git tag, branch, or commit SHA.
- Download a single file, a subset of files, or all files from the repo.
- Download one or more binary assets from a specific release that match a regular expression.
- Verify the SHA256 or SHA512 checksum of a binary asset.
- Download from public repos, or from private repos by specifying a [GitHub Personal Access Token](https://help.github.com/articles/creating-an-access-token-for-command-line-use/).
- Download from GitHub Enterprise.
- When specifying a git tag, you can can specify either exactly the tag you want, or a [Tag Constraint Expression](#tag-constraint-expressions) to do things like "get the latest non-breaking version" of this repo. Note that fetch assumes git tags are specified according to [Semantic Versioning](http://semver.org/) principles.

#### Quick examples

Download folder `/baz` from tag `0.1.3` of a GitHub repo and save it to `/tmp/baz`:

```
fetch --repo="https://github.com/foo/bar" --tag="0.1.3" --source-path="/baz" /tmp/baz
```

Download a release asset matching named `foo.exe` from release `0.1.5` and save them to `/tmp`:

```
fetch --repo="https://github.com/foo/bar" --tag="0.1.5" --release-asset="foo.exe" /tmp
```

Download all release assets matching the regular expression, `foo_linux-.*` from release `0.1.5` and save them to `/tmp`:

```
fetch --repo="https://github.com/foo/bar" --tag="0.1.5" --release-asset="foo_linux-.*" /tmp
```

See more examples in the [Examples section](#examples).

## Installation

### Download from releases page

Download the fetch binary from the [GitHub Releases](https://github.com/gruntwork-io/fetch/releases) tab.

### Install via package manager

Note that package managers are third party. The third party fetch packages may not be updated with the latest version, but are often close. Please check your version against the latest available on the [releases page](https://github.com/gruntwork-io/fetch/releases). If you want the latest version, the recommended installation option is to [download from the releases page](https://github.com/gruntwork-io/fetch/releases).

- **macOS:** You can install fetch using [Homebrew](https://brew.sh/): `brew install fetch`. 

- **Linux:** Most Linux users can use [Homebrew](https://docs.brew.sh/Homebrew-on-Linux): `brew install fetch`.

## Usage

#### Assumptions

fetch assumes that a repo's tags are in the format `vX.Y.Z` or `X.Y.Z` to support Semantic Versioning parsing. This allows you to specify a [Tag Constraint Expression](#tag-constraint-expressions) to do things like "get the latest non-breaking version" of this repo. Note that fetch also allows downloading a specific tag not in SemVer format.

#### General Usage

```
fetch [OPTIONS] <local-download-path>
```

The supported options are:

- `--repo` (**Required**): The fully qualified URL of the GitHub repo to download from (e.g. https://github.com/foo/bar).
- `--ref` (**Optional**): The git reference to download. If specified, will override `--commit`, `--branch`, and `--tag`.
- `--tag` (**Optional**): The git tag to download. Can be a specific tag or a [Tag Constraint
  Expression](#tag-constraint-expressions).
- `--branch` (**Optional**): The git branch from which to download; the latest commit in the branch will be used. If
  specified, will override `--tag`.
- `--commit` (**Optional**): The SHA of a git commit to download. If specified, will override `--branch` and `--tag`.
- `--source-path` (**Optional**): The source path to download from the repo (e.g. `--source-path=/folder` will download
  the `/folder` path and all files below it). By default, all files are downloaded from the repo unless `--source-path`
  or `--release-asset` is specified. This option can be specified more than once.
- `--release-asset` (**Optional**): A regular expression matching release assets--these are binary files uploaded to a [GitHub
  Release](https://help.github.com/articles/creating-releases/)--to download. It only works with the `--tag` option.
- `--release-asset-checksum` (**Optional**): The checksum that a release asset should have. Fetch will fail if this value
  is non-empty and does not match the checksum computed by Fetch, or if more than 1 assets are matched by the release-asset
  regular expression.
- `--release-asset-checksum-algo` (**Optional**): The algorithm fetch will use to compute a checksum of the release asset.
  Supported values are `sha256` and `sha512`.
- `--github-oauth-token` (**Optional**): A [GitHub Personal Access
  Token](https://help.github.com/articles/creating-an-access-token-for-command-line-use/). Required if you're
  downloading from private GitHub repos. **NOTE:** fetch will also look for this token using the `GITHUB_OAUTH_TOKEN`
  environment variable, which we recommend using instead of the command line option to ensure the token doesn't get
  saved in bash history.
- `--github-api-version` (**Optional**): Used when fetching an artifact from a GitHub Enterprise instance.
  Defaults to `v3`. This is ignored when fetching from GitHub.com.
- `--progress` (**Optional**): Used when fetching a big file and want to see progress on the fetch.

The supported arguments are:

- `<local-download-path>` (**Required**): The local path where all files should be downloaded (e.g. `/tmp`).

Run `fetch --help` to see more information about the flags.

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

## Examples

#### Usage Example 1

Download `/modules/foo/bar.sh` from a GitHub release where the tag is the latest version of `0.1.x` but at least `0.1.5`, and save it to `/tmp/bar`:

```
fetch --repo="https://github.com/foo/bar" --tag="~>0.1.5" --source-path="/modules/foo/bar.sh" /tmp/bar
```

#### Usage Example 2

Download all files in `/modules/foo` from a GitHub release where the tag is exactly `0.1.5`, and save them to `/tmp`:

```
fetch --repo="https://github.com/foo/bar" --ref="0.1.5" --source-path="/modules/foo" /tmp
```

#### Usage Example 3

Download all files from a private GitHub repo using the GitHUb oAuth Token `123`. Get the release whose tag is exactly `0.1.5`, and save the files to `/tmp`:

```
GITHUB_OAUTH_TOKEN=123

fetch --repo="https://github.com/foo/bar" --ref="0.1.5" /tmp
```

#### Usage Example 4

Download all files from the latest commit on the `sample-branch` branch, and save them to `/tmp`:

```
fetch --repo="https://github.com/foo/bar" --ref="sample-branch" /tmp/josh1
```

#### Usage Example 5

Download all files from the git commit `f32a08313e30f116a1f5617b8b68c11f1c1dbb61`, and save them to `/tmp`:

```
fetch --repo="https://github.com/foo/bar" --ref="f32a08313e30f116a1f5617b8b68c11f1c1dbb61" /tmp
```

#### Usage Example 6

Download the release asset `foo.exe` from a GitHub release where the tag is exactly `0.1.5`, and save it to `/tmp`:

```
fetch --repo="https://github.com/foo/bar" --ref="0.1.5" --release-asset="foo.exe" /tmp
```

#### Usage Example 7

Download the release asset `foo.exe` from a GitHub release hosted on a GitHub Enterprise instance running at `ghe.mycompany.com` where the tag is exactly `0.1.5`, and save it to `/tmp`:

```
fetch --repo="https://ghe.mycompany.com/foo/bar" --ref="0.1.5" --release-asset="foo.exe" /tmp
```

##### Release Instructions

To release a new version of `fetch`, go to the [Releases page](https://github.com/gruntwork-io/fetch/releases) and "Draft a new release".
On the following page, bump the "Tag version" appropriately, and set the "Release title" to be the same.
In the "Describe this release" section, log the changes of this release, linking back to issues that were addressed.
Click the "Publish release" button. CircleCI will pick this up, generate the assets, and attach them to the release.

## License

This code is released under the MIT License. See [LICENSE.txt](/LICENSE.txt).

## TODO

- Introduce code verification using something like GPG signatures or published checksums
- Explicitly test for exotic repo and org names
- Apply stricter parsing for repo-filter command-line arg