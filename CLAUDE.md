# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## YOU MUST

- Answers should be in Japanese.
- TODOs should always include creating a branch, testing the implementation, committing
- Whenever you modify a feature, be sure to modify (or create) a test and make sure the test passes!

## Repository Configuration

- **Author**: FrontWorksDev
- **Repository Name**: Loki
- **MCP GitHub API**: always use `Loki` repository

## Be sure to do this at the beginning and end of the work when adding a modification feature. Be sure to include everything in the todo each time

- The following operations must be performed at the start of work
  - **At the start**: Always create a dedicated branch (feature/<feature name>, fixed/<fix description>, etc.)
  - **NEVER work directly on the main branch**: Never commit any changes directly to the main branch.
- Be sure to perform the following at the end of the work
  1. create commit

## Project Overview

**Loki** is a Go project for image compression. The project uses Go 1.25.6 managed by asdf.

## Development Setup

### Prerequisites

- asdf (version manager)
- Go 1.25.6 (managed by asdf via `.tool-versions`)

### Getting Started

1. Ensure asdf is installed: `asdf --version`
2. Install the golang plugin if not already installed: `asdf plugin add golang` (if needed)
3. Install Go 1.25.6: `asdf install`
4. Install development tools:
   - gopls (Go language server)
   - delve (debugger)
   - goimports (import formatter)
   - golangci-lint (linter)

## Common Development Commands

### Building

```bash
go build ./...                      # Build all packages
go build -o build/img-cli ./cmd/img-cli  # Build CLI to build directory
```

### Testing

```bash
go test ./...                                          # Run all tests
go test -v ./...                                       # Run tests with verbose output
go test -run TestName ./...                            # Run specific test
go test -cover ./...                                   # Run tests with coverage
go test ./... -coverprofile=coverage.out -covermode=atomic  # Generate coverage profile (Codecov compatible)
go tool cover -html=coverage.out                       # View coverage in browser
go tool cover -func=coverage.out                       # Show coverage per function
```

### Linting & Formatting

```bash
golangci-lint run ./...          # Run linter checks
go fmt ./...                      # Format code
goimports -w ./...               # Format imports
```

### Running

```bash
go run ./cmd/...                 # Run main package
```

### Debugging

- Use delve debugger in your IDE
- Or run: `dlv debug ./cmd/...`

## Project Structure

```
Loki/
├── cmd/
│   └── img-cli/          # CLI application for image processing
├── internal/
│   └── imageproc/        # Image processing packages (resize, etc.)
├── testdata/             # Test fixtures and sample data
│   └── output/          # Test output files (gitignored)
├── build/               # Build artifacts (gitignored)
├── configs/             # Configuration files
└── scripts/             # Build and utility scripts
```

## Development Guidelines

### Go Best Practices

- Follow standard Go naming conventions (camelCase for unexported, PascalCase for exported)
- Use `go fmt` and `goimports` before committing
- Run `golangci-lint` to catch common issues
- Write tests alongside implementation code

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
