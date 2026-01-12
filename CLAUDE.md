# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## YOU MUST

- Answers should be in Japanese.
- TODOs should always include creating a branch, testing the implementation, committing
- Whenever you modify a feature, be sure to modify (or create) a test and make sure the test passes!

## Project Overview

**image-compresser** is a Go project for image compression. The project uses Go 1.24 with a development container setup including neovim, gopls, delve debugger, goimports, air (hot reload), and golangci-lint.

## Development Setup

### Prerequisites

- Docker and Dev Containers support (VSCode, Cursor, etc.)
- The project is configured to use a devcontainer with Go 1.24 pre-installed

### Getting Started

1. Open the project in a Dev Container environment
2. The container will automatically install:
   - gopls (Go language server)
   - delve (debugger)
   - goimports (import formatter)
   - air (hot reload for development)
   - golangci-lint (linter)

## Common Development Commands

### Building

```bash
go build ./...          # Build all packages
go build -o bin/app     # Build with output binary
```

### Testing

```bash
go test ./...                    # Run all tests
go test -v ./...                 # Run tests with verbose output
go test -run TestName ./...      # Run specific test
go test -cover ./...             # Run tests with coverage
go test -coverprofile=coverage.out ./...  # Generate coverage profile
go tool cover -html=coverage.out      # View coverage in browser
```

### Linting & Formatting

```bash
golangci-lint run ./...          # Run linter checks
go fmt ./...                      # Format code
goimports -w ./...               # Format imports
```

### Running

```bash
go run main.go                   # Run main package
air                              # Run with hot reload (development)
```

### Debugging

- Use delve debugger in your IDE
- Or run: `dlv debug ./...`

## Project Structure

To be completed as the project grows. Initially, expect:

- `main.go` - Entry point
- Supporting packages for image compression logic
- `testdata/` - Test fixtures and sample data

## Development Guidelines

### Go Best Practices

- Follow standard Go naming conventions (camelCase for unexported, PascalCase for exported)
- Use `go fmt` and `goimports` before committing
- Run `golangci-lint` to catch common issues
- Write tests alongside implementation code

### Hot Reload

Use `air` during development for automatic recompilation:

```bash
air
```

This watches for file changes and rebuilds the application automatically.

### Testing Strategy

- Place tests in the same package as the code being tested
- Name test files `*_test.go`
- Use table-driven tests for multiple test cases

### Commit Messages

- Use Japanese for commit messages (日本語でコミットメッセージを書く)
- Keep title under 50 characters (タイトルは50文字以内)
- Include purpose and expected impact (変更の目的と期待される影響を含める)
- Use natural line breaks, not `\n` (改行に`\n`は使用せず通常の改行)
- Include main changes and feature additions when applicable:
  - **主な変更点：**
  - **機能追加**
