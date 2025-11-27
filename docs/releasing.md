# Release Process

This document describes the process for building and releasing Kilt binaries.

## Overview

Kilt uses [GoReleaser](https://goreleaser.com) to automate the build and release process. When a Git tag is pushed, GitHub Actions automatically builds binaries for all supported platforms and creates a GitHub release.

## Supported Platforms

- **macOS**: amd64 (Intel), arm64 (Apple Silicon)
- **Linux**: amd64, arm64

## Prerequisites

- Go 1.21 or later
- [GoReleaser](https://goreleaser.com/install/) (for local testing)
- GitHub repository with Actions enabled
- Write access to the repository for creating releases

## Release Process

### 1. Prepare Release

Before creating a release, ensure:

- All tests pass: `make test`
- Code is linted: `make lint`
- Version is updated in relevant files (if needed)
- CHANGELOG is updated (optional, GoReleaser can generate from Git tags)

### 2. Create Git Tag

Create and push a Git tag with a semantic version:

```bash
# Create annotated tag
git tag -a v1.0.0 -m "Release v1.0.0"

# Push tag to trigger release workflow
git push origin v1.0.0
```

**Important**: Tag names must start with `v` (e.g., `v1.0.0`, `v1.2.3-beta.1`).

### 3. Automated Release

When the tag is pushed:

1. GitHub Actions workflow (`.github/workflows/release.yml`) is triggered
2. GoReleaser builds binaries for all platforms
3. Checksums (SHA256) are generated for all assets
4. GitHub release is created with:
   - Release notes (auto-generated from Git commits)
   - All platform binaries (as tar.gz archives)
   - `checksums.txt` file for verification
5. Release is published to GitHub

### 4. Verify Release

After the release is created:

1. Check the [GitHub Releases page](https://github.com/unravelling/kilt/releases)
2. Verify all platform binaries are present
3. Verify checksums file is included
4. Test the installer script with the new release

## Local Testing

You can test the release process locally using GoReleaser:

```bash
# Install GoReleaser (if not already installed)
go install github.com/goreleaser/goreleaser@latest

# Test the configuration (dry-run)
goreleaser release --snapshot --skip-publish

# This creates binaries in ./dist/ directory
```

## Release Assets

Each release includes:

- **Archives**: `kilt-{version}-{os}-{arch}.tar.gz` containing:
  - `kilt` binary
  - `LICENSE`
  - `README.md`
- **Checksums**: `checksums.txt` with SHA256 hashes for all assets

## Version Injection

Version information is injected into binaries at build time:

- `version`: Git tag (e.g., `v1.0.0`)
- `commit`: Git commit SHA
- `date`: Build timestamp

This information is available via `kilt version` command.

## Configuration Files

- **`.goreleaser.yml`**: GoReleaser configuration
- **`.github/workflows/release.yml`**: GitHub Actions release workflow

## Troubleshooting

### Release Workflow Fails

1. Check GitHub Actions logs for errors
2. Verify GoReleaser configuration is valid: `goreleaser check`
3. Ensure Git tag format is correct (must start with `v`)
4. Verify repository has `contents: write` permission

### Missing Binaries

1. Check build logs for compilation errors
2. Verify all platforms are specified in `.goreleaser.yml`
3. Ensure Go version supports all target platforms

### Checksum Verification Fails

1. Verify `checksums.txt` is generated correctly
2. Check that installer script uses correct checksum algorithm (SHA256)
3. Ensure checksums file is uploaded to release

## Homebrew Tap (Optional)

To enable Homebrew installation:

1. Create a Homebrew tap repository: `unravelling/homebrew-tap`
2. Uncomment the `brews` section in `.goreleaser.yml`
3. Add `HOMEBREW_TAP_GITHUB_TOKEN` secret to GitHub repository
4. GoReleaser will automatically create/update the Homebrew formula

## Manual Release (Fallback)

If automated release fails, you can create a manual release:

```bash
# Build all binaries locally
make build-all

# Create GitHub release manually
# Upload binaries from bin/ directory
# Include checksums.txt
```

## Release Checklist

- [ ] All tests pass
- [ ] Code is linted and formatted
- [ ] Version updated (if needed)
- [ ] Git tag created and pushed
- [ ] GitHub Actions workflow completed successfully
- [ ] Release assets verified
- [ ] Installer script tested with new release
- [ ] Documentation updated (if needed)

## Related Documentation

- [Installation Guide](../README.md#quick-start)
- [Architecture Documentation](architecture.md)
- [GoReleaser Documentation](https://goreleaser.com)

