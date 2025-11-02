# nanolayer-go

A Go-based CLI tool for working with development containers.

## Installation

Download the latest binary from the [releases page](https://github.com/devcontainer-community/nanolayer-go/releases).

## Building from Source

```bash
go build -o nanolayer .
```

## Usage

### Available Commands

- `nanolayer test` - Run a test command
- `nanolayer version` - Display version information
- `nanolayer --version` - Display version information (shorthand)

### Examples

```bash
# Run the test command
./nanolayer test

# Check version
./nanolayer version
./nanolayer --version
```

## Development

### Prerequisites

- Go 1.25 or later

### Building with Version Information

```bash
go build -ldflags "\
  -X github.com/devcontainer-community/nanolayer-go/cmd.Version=v1.0.0 \
  -X github.com/devcontainer-community/nanolayer-go/cmd.Commit=$(git rev-parse HEAD) \
  -X github.com/devcontainer-community/nanolayer-go/cmd.Date=$(date -u +%Y-%m-%dT%H:%M:%SZ)" \
  -o nanolayer .
```

## Releasing

This project uses [GoReleaser](https://goreleaser.com/) for building and releasing binaries.

To create a new release:

1. Tag your commit: `git tag -a v1.0.0 -m "Release v1.0.0"`
2. Push the tag: `git push origin v1.0.0`
3. GitHub Actions will automatically build and release binaries for Linux (amd64 and arm64)

### Testing GoReleaser Locally

```bash
# Install GoReleaser
go install github.com/goreleaser/goreleaser@latest

# Test the release process (without publishing)
goreleaser release --snapshot --clean
```

## License

[Add your license here]
