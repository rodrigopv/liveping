# Release Process

This document outlines the process for creating new releases of the `liveping` tool.

## Overview

The release process is automated using [GitHub Actions](https://github.com/features/actions) and [GoReleaser](https://goreleaser.com/). When a specially named branch is pushed **directly to this repository**, a workflow automatically builds the application for multiple platforms, creates a draft GitHub release, and attaches the compiled binaries and checksums.

## Triggering a Release

To trigger the release workflow, **a maintainer with push access** should create and push a branch to **this repository** that follows this naming convention:

```
release/vX.Y.Z
```

Where `X.Y.Z` represents the semantic version number for the release (e.g., `v1.0.0`, `v1.1.0-beta.1`).

**Example (for maintainers):**

```bash
# Assuming you are on the main branch with the latest changes

# Create the release branch
git checkout -b release/v1.0.0

# Push the release branch to the main repository on GitHub
# (This requires maintainer permissions)
git push origin release/v1.0.0
```

Pushing this branch *directly to the repository* will automatically start the "Release Build" workflow defined in `.github/workflows/release.yml`.

**Note for Contributors:** Pull requests from forks with branches named `release/*` will **not** trigger this release workflow. This process is intended for maintainers initiating an official release.

## Automated Workflow Steps

1.  **Trigger**: The workflow starts when a `release/v*` branch is pushed **directly to this repository**.
2.  **Checkout**: The workflow checks out the code from the pushed branch.
3.  **Setup Go**: It sets up the required Go environment (version 1.21 as specified in the workflow).
4.  **Extract Version**: It extracts the version number (e.g., `v1.0.0`) from the branch name.
5.  **Run GoReleaser**: It executes the `goreleaser/goreleaser-action`.
    *   GoReleaser reads the configuration from `.goreleaser.yml`.
    *   **Build**: It builds the `liveping` binary for:
        *   Linux (amd64, arm64)
        *   macOS (amd64, arm64)
        *   Windows (amd64)
    *   **Archive**: It creates archives (`.tar.gz` for Linux/macOS, `.zip` for Windows) containing the binary, `LICENSE`, and `README.md` files.
    *   **Checksum**: It generates a `checksums.txt` file containing SHA256 checksums for all generated archives.
    *   **Create Draft Release**: It uses the extracted version to tag a new release and creates a **draft** release on the GitHub repository.
    *   **Upload Assets**: It uploads the generated archives and the `checksums.txt` file to the draft release.

## Manual Final Step: Publishing the Release

After the workflow completes successfully:

1.  Navigate to the **Releases** section of the GitHub repository.
2.  You will find a new **draft** release tagged with the version you pushed (e.g., `v1.0.0`).
3.  Review the release notes (which are initially just a placeholder like "Draft release for v1.0.0"). **Edit the release notes** to include details about the changes in this version.
4.  Verify the attached assets (binaries, checksums).
5.  Once you are satisfied, **publish** the release.

This makes the release official and publicly available. 