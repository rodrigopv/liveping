# Release Process

This document outlines the process for creating new releases of the `liveping` tool.

## Overview

The release process is automated using [GitHub Actions](https://github.com/features/actions) and [GoReleaser](https://goreleaser.com/). When a new Git tag matching the pattern `v*.*.*` is pushed **to this repository**, a workflow automatically builds the application for multiple platforms, creates a draft GitHub release, and attaches the compiled binaries and checksums.

## Triggering a Release

To create a new release, **a maintainer with push access** should perform the following steps:

1.  Ensure the `main` branch (or the branch you release from) is up-to-date and contains all changes intended for the release.
2.  Create a new Git tag matching the pattern `vX.Y.Z`, where `X.Y.Z` represents the semantic version number (e.g., `v0.1.0`, `v1.0.0`).
3.  Push the tag **to this repository**.

**Example (for maintainers):**

```bash
# Ensure you are on the correct branch (e.g., main) and it's up-to-date
git checkout main
git pull origin main

# Create the new tag
# (Use -a for an annotated tag, which is recommended)
git tag -a v0.1.0 -m "Release version 0.1.0"

# Push the tag to the main repository on GitHub
# (This requires maintainer permissions)
git push origin v0.1.0
```

Pushing this tag *directly to the repository* will automatically start the "Release Build" workflow defined in `.github/workflows/release.yml`.

## Automated Workflow Steps

1.  **Trigger**: The workflow starts when a `v*.*.*` tag is pushed **directly to this repository**.
2.  **Checkout**: The workflow checks out the code corresponding to the pushed tag.
3.  **Setup Go**: It sets up the required Go environment (version 1.24 as specified in the workflow).
4.  **Run GoReleaser**: It executes the `goreleaser/goreleaser-action`.
    *   GoReleaser automatically detects the version from the pushed Git tag.
    *   GoReleaser reads the configuration from `.goreleaser.yml`.
    *   **Build**: It builds the `liveping` binary for:
        *   Linux (amd64, arm64)
        *   macOS (amd64, arm64)
        *   Windows (amd64)
    *   **Archive**: It creates archives (`.tar.gz` for Linux/macOS, `.zip` for Windows) containing the binary, `LICENSE`, and `README.md` files.
    *   **Checksum**: It generates a `checksums.txt` file containing SHA256 checksums for all generated archives.
    *   **Changelog/Notes**: By default (or based on `.goreleaser.yml` settings), it may generate release notes from commit messages since the last tag.
    *   **Create Draft Release**: It uses the tag version to create a **draft** release on the GitHub repository.
    *   **Upload Assets**: It uploads the generated archives and the `checksums.txt` file to the draft release.

## Manual Final Step: Publishing the Release

After the workflow completes successfully:

1.  Navigate to the **Releases** section of the GitHub repository.
2.  You will find a new **draft** release tagged with the version you pushed (e.g., `v0.1.0`).
3.  Review the automatically generated release notes. **Edit or augment them** as needed to provide clear information about the changes in this version.
4.  Verify the attached assets (binaries, checksums).
5.  Once you are satisfied, **publish** the release.

This makes the release official and publicly available. 