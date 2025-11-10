# Contributing to ArmyKnife CLI

We love contributions! Here's how you can help.

## Getting Started

1. **Fork the repository** on GitHub
2. **Clone your fork** locally:
   ```bash
   git clone https://github.com/YOUR_USERNAME/armyknife-cli
   cd armyknife-cli
   ```
3. **Create a feature branch**:
   ```bash
   git checkout -b feature/your-feature-name
   ```

## Development Setup

### Prerequisites

- Go 1.21 or later
- Git

### Building

```bash
# Install dependencies
go mod download

# Build the CLI
go build -o armyknife ./cmd

# Run the CLI
./armyknife --help
```

### Testing

```bash
# Run all tests
go test ./...

# Run tests with verbose output
go test -v ./...

# Run tests with coverage
go test -cover ./...
```

## Code Style

We follow standard Go conventions:

- **Formatting**: Use `gofmt` or `go fmt`
- **Linting**: Run `golangci-lint` before submitting
- **Comments**: Include comments for exported functions
- **Error Handling**: Always handle errors appropriately

### Before Submitting

```bash
# Format code
go fmt ./...

# Run linter
golangci-lint run

# Run tests
go test ./...

# Build for all platforms
GOOS=linux GOARCH=amd64 go build -o armyknife-linux-amd64
GOOS=darwin GOARCH=amd64 go build -o armyknife-darwin-amd64
GOOS=windows GOARCH=amd64 go build -o armyknife-windows-amd64.exe
```

## Commit Messages

Write clear, descriptive commit messages:

```
feat(code): add hybrid search functionality

- Implement vector + keyword scoring
- Add weighting configuration
- Update tests and documentation

Closes #123
```

### Commit Types

- `feat`: A new feature
- `fix`: A bug fix
- `docs`: Documentation changes
- `style`: Code style changes (formatting, linting)
- `refactor`: Code refactoring
- `perf`: Performance improvements
- `test`: Test additions or changes
- `chore`: Build process, dependencies, tooling

## Pull Request Process

1. **Update the README** if you change functionality
2. **Add tests** for new code
3. **Update documentation** as needed
4. **Create a pull request** with:
   - Clear title describing the change
   - Description of what changed and why
   - Reference to related issues (if any)
   - Tests for new functionality

### PR Title Format

```
[TYPE] Brief description

Examples:
- [FEAT] Add hybrid search with configurable weights
- [FIX] Fix query latency measurement for p99
- [DOCS] Update installation instructions for Windows
```

## Architecture

### Directory Structure

```
armyknife-cli/
├── cmd/              # Command-line interface
├── internal/         # Private packages
│   ├── client/       # API client implementation
│   ├── config/       # Configuration management
│   └── types/        # Type definitions
├── pkg/              # Public packages
│   └── output/       # Output formatting
└── main.go           # Entry point
```

### Key Components

- **cmd/**: Command definitions and argument parsing
- **internal/client/**: HTTP client for Code Intelligence API
- **internal/config/**: Configuration from environment variables
- **pkg/output/**: Formatted output for CLI

## Testing Guidelines

- Write tests for new features
- Update tests when changing existing functionality
- Aim for >80% code coverage
- Test edge cases and error conditions

Example test:

```go
func TestQuery(t *testing.T) {
    // Setup
    client := NewTestClient()

    // Test
    results, err := client.Query(context.Background(), "test query")

    // Assert
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    if len(results) == 0 {
        t.Fatal("expected results, got none")
    }
}
```

## Documentation

- Update README.md for user-facing changes
- Add inline comments for complex logic
- Update API documentation if needed
- Include examples for new features

## Issues

Before starting work, check if there's an open issue:

1. **Search existing issues** to avoid duplicates
2. **Create a new issue** if one doesn't exist
3. **Link your PR** to the issue

## Release Process

New releases are automatically created when tags are pushed:

```bash
# Create a new version tag (maintainers only)
git tag -a v1.1.0 -m "Release v1.1.0"
git push origin v1.1.0
```

GitHub Actions will automatically:
- Build binaries for all platforms
- Create SHA256 checksums
- Create a GitHub Release
- Publish documentation

## Questions?

- **Issues**: [GitHub Issues](https://github.com/armyknifelabs-platform/armyknife-cli/issues)
- **Discussions**: [GitHub Discussions](https://github.com/armyknifelabs-platform/armyknife-cli/discussions)

## Code of Conduct

Be respectful and constructive in all interactions. We're here to help each other build great software.

---

Thank you for contributing to ArmyKnife! 🎉
