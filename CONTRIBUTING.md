# Contributing to otelfuego

Thank you for your interest in contributing to otelfuego! This document provides guidelines and information for contributors.

## Development Environment

### Prerequisites

- Go 1.23+ (see [go.mod](go.mod) for minimum version)
- Git

### Setup

1. Fork the repository on GitHub
2. Clone your fork locally:
   ```bash
   git clone https://github.com/your-username/otelfuego.git
   cd otelfuego
   ```
3. Install dependencies:
   ```bash
   go mod download
   ```

## Development Workflow

### Running Tests

```bash
# Run all tests
go test ./...

# Run tests with race detection
go test -race ./...

# Run tests with coverage
go test -cover ./...

# Run tests with verbose output
go test -v ./otelfuego
```

### Code Quality

Before submitting changes, ensure your code passes:

```bash
# Format code
go fmt ./...

# Run static analysis
go vet ./...

# Build all packages
go build ./...
```

### Code Style

- Follow standard Go conventions and idioms
- Use `gofmt` to format your code
- Add comments for exported functions and types
- Write clear, descriptive commit messages

## Submitting Changes

### Pull Request Process

1. Create a feature branch from `main`:
   ```bash
   git checkout -b feature/your-feature-name
   ```

2. Make your changes and add tests
3. Ensure all tests pass and code is properly formatted
4. Commit your changes with a descriptive message
5. Push to your fork and create a pull request

### Pull Request Guidelines

- Provide a clear description of the changes
- Reference any related issues
- Include tests for new functionality
- Update documentation if needed
- Ensure CI checks pass

## Reporting Issues

### Bug Reports

When reporting bugs, please include:

- Go version and operating system
- Minimal code example that reproduces the issue
- Expected vs actual behavior
- Full error messages and stack traces

### Feature Requests

For new features:

- Describe the use case and motivation
- Provide examples of how the feature would be used
- Consider backwards compatibility

## Code Organization

- `otelfuego/fuego.go` - Core middleware implementation
- `otelfuego/config.go` - Configuration and options
- `otelfuego/example_test.go` - Tests and usage examples

## Questions?

If you have questions about contributing, feel free to:
- Open an issue for discussion
- Check existing issues and discussions

Thank you for contributing to otelfuego!