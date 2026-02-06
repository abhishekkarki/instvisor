# Contributing to Instvisor

Thank you for your interest in contributing! 

## Code of Conduct

This project adheres to a code of conduct. By participating, you are expected to uphold this code.

## How Can I Contribute?

### Reporting Bugs

Before creating bug reports, please check existing issues. When creating a bug report, include:

- **Description**: Clear description of the bug
- **Steps to Reproduce**: Detailed steps
- **Expected Behavior**: What you expected to happen
- **Actual Behavior**: What actually happened
- **Environment**: OS, Go version, Instvisor version
- **Logs**: Relevant log output

### Suggesting Enhancements

Enhancement suggestions are tracked as GitHub issues. When creating an enhancement suggestion:

- **Use a clear title**
- **Provide detailed description** of the suggested enhancement
- **Explain why this would be useful** to most users
- **List examples** of how it would be used

### Pull Requests

1. Fork the repo and create your branch from `main`
2. If you've added code, add tests
3. Ensure the test suite passes
4. Make sure your code follows Go conventions (`gofmt`, `golint`)
5. Write a clear commit message

## Development Workflow

### Setup
```bash
git clone https://github.com/yourusername/instvisor.git
cd instvisor
go mod download
```

### Building
```bash
make build          # Build binaries
make build-all      # Build all tools
make test           # Run tests
make clean          # Clean build artifacts
```

### Code Style

- Follow standard Go conventions
- Run `gofmt` before committing
- Keep functions focused and small
- Add comments for exported functions
- Use meaningful variable names

### Testing
```bash
# Run all tests
go test ./...

# Run specific package
go test ./pkg/collector

# With coverage
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

### Commit Messages

- Use the present tense ("Add feature" not "Added feature")
- Use the imperative mood ("Move cursor to..." not "Moves cursor to...")
- Limit first line to 72 characters
- Reference issues and pull requests

Example:
```
Add Prometheus exporter endpoint

- Implement /metrics endpoint
- Export CPU, memory, disk, network metrics
- Add configuration for exporter port

Fixes #42
```

## Project Structure
```
instvisor/
├── cmd/
│   ├── agent/          # Main agent binary
│   └── analyze/        # Analysis CLI tool
├── pkg/
│   ├── collector/      # Metric collectors
│   ├── storage/        # Storage implementations
│   ├── analyzer/       # Analysis engine
│   ├── metrics/        # Metric types
│   └── config/         # Configuration
├── configs/            # Default configurations
├── scripts/            # Installation scripts
├── deployments/        # Docker, K8s configs
└── docs/               # Documentation
```

## Adding a New Collector

Example: Adding a GPU collector

1. Create `pkg/collector/gpu.go`:
```go
package collector

type GPUCollector struct {
    interval time.Duration
}

func NewGPUCollector(interval time.Duration) *GPUCollector {
    return &GPUCollector{interval: interval}
}

func (g *GPUCollector) Name() string {
    return "gpu"
}

func (g *GPUCollector) Interval() time.Duration {
    return g.interval
}

func (g *GPUCollector) Collect() ([]metrics.Metric, error) {
    // Implementation
}
```

2. Update `pkg/collector/manager.go` to register it
3. Add configuration in `pkg/config/config.go`
4. Add tests in `pkg/collector/gpu_test.go`
5. Update documentation

## Questions?

Feel free to ask in [Discussions](https://github.com/abhishekkarki/instvisor/discussions).
```
