# Development Guide

## Prerequisites

- Go 1.25 or later
- Docker daemon running
- Make (optional, for build automation)

## Setup

```bash
# Clone repository
cd scripts/tools/stats

# Download dependencies
go mod download

# Verify setup
go vet ./...
```

## Development Workflow

### Building

```bash
# Quick build
go build -o docker-stats .

# Production build (optimized)
make build

# Build for all platforms
make build-all
```

### Testing

```bash
# Run all tests
make test

# Run tests with coverage
make test-coverage

# Run specific test
go test -v -run TestFormatBytes ./internal/docker/
```

### Code Quality

```bash
# Format code
make fmt

# Run linter
make lint

# Security scan
make security

# All checks
make all
```

## Code Style

### Naming Conventions

- Use camelCase for variables and functions
- Use PascalCase for exported types and functions
- Prefix unexported functions with lowercase

### Error Handling

```go
// Good
if err != nil {
    return fmt.Errorf("failed to get stats: %w", err)
}

// Bad
if err != nil {
    return err
}
```

### Comments

```go
// FormatBytes formats bytes into human-readable format.
// It uses binary prefixes (KiB, MiB, GiB, etc.).
func FormatBytes(bytes uint64) string {
    // ...
}
```

## Adding New Features

### Adding a New Column

1. Add field to `ContainerStats` struct in `client.go`
2. Populate field in `getContainerStats()`
3. Add column header in `updateTable()`
4. Add cell rendering in the data loop

### Adding a New Sort Field

1. Add constant to `SortField` enum in `client.go`
2. Add case to `SortContainers()` switch
3. Add keyboard shortcut in `handleInput()`
4. Update status bar text

### Adding a New Keyboard Shortcut

1. Add case to `handleInput()` in `app.go`
2. Update status bar text
3. Document in README.md

## Testing Guidelines

### Unit Tests

```go
func TestFormatBytes(t *testing.T) {
    tests := []struct {
        name     string
        bytes    uint64
        expected string
    }{
        {"zero bytes", 0, "0B"},
        {"kilobytes", 1024, "1.0KiB"},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result := FormatBytes(tt.bytes)
            if result != tt.expected {
                t.Errorf("got %s; want %s", result, tt.expected)
            }
        })
    }
}
```

### Integration Tests

Integration tests require Docker daemon:

```go
func TestDockerClient(t *testing.T) {
    if testing.Short() {
        t.Skip("skipping integration test")
    }
    
    client, err := NewClient()
    if err != nil {
        t.Fatalf("failed to create client: %v", err)
    }
    defer client.Close()
    
    // Test operations...
}
```

## Debugging

### Enable Verbose Logging

```go
import "log"

log.Printf("Container: %s, CPU: %.2f%%", cont.Name, cont.CPUPercent)
```

### Docker API Debugging

```bash
# Check Docker API version
docker version

# Test Docker connectivity
docker ps

# View Docker events
docker events
```

## Release Process

1. Update version in `main.go`
2. Update CHANGELOG.md
3. Run all checks: `make all`
4. Build for all platforms: `make build-all`
5. Tag release: `git tag v1.x.x`
6. Push tag: `git push --tags`

## Troubleshooting

### "Cannot connect to Docker daemon"

```bash
# Check Docker status
sudo systemctl status docker

# Start Docker
sudo systemctl start docker

# Check socket permissions
ls -la /var/run/docker.sock
```

### "Permission denied"

```bash
# Add user to docker group
sudo usermod -aG docker $USER

# Apply changes (logout/login or)
newgrp docker
```

### Build Errors

```bash
# Clear module cache
go clean -modcache

# Re-download dependencies
go mod download

# Verify dependencies
go mod verify
```
