# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## YOU MUST

- Answers should be in Japanese.
- TODOs should always include creating a branch, testing the implementation, committing, pushing, and creating a PR (if not already done)
- Whenever you modify a feature, be sure to modify (or create) a test and make sure the test passes!

## Repository Configuration

- **Author**: FrontWorksDev
- **Repository Name**: image-compressor
- **MCP GitHub API**: always use `image-compressor` repository

## Build and Development Commands

```bash
# Build the binary (output: ./build/imgcompress)
make build

# Run all tests with verbose output
make test

# Run a single test
go test -v -run TestFunctionName ./internal/...

# Test coverage report (generates coverage.html)
make test-coverage

# Run linter
make lint

# Install to GOPATH
make install

# Clean build artifacts
make clean
```

## Architecture

This is a Go CLI tool for compressing JPEG/PNG images.

**Data flow**: `main.go` → `cli.Parse()` → `image.Processor.Load()` → `compressor.GetCompressor()` → `Compressor.Compress()` → write to file

### Key Components

- **`internal/cli`**: CLI argument parsing with `flag` package. `Options` struct holds quality, suffix, verbose, and input paths.
- **`internal/image`**: `Processor` handles file I/O and format detection via magic bytes (not file extension). Also provides `QualityToPNGCompression()` for quality-to-compression-level mapping.
- **`internal/compressor`**: `Compressor` interface with `JPEGCompressor` and `PNGCompressor` implementations. Factory pattern via `GetCompressor(format)`.

### Format Detection

Images are detected by magic bytes in `processor.go:detectFormat()`:
- JPEG: `0xFF 0xD8`
- PNG: `0x89 0x50 0x4E 0x47`

## Git Workflow

### At the Start of Work

- **Always create a dedicated branch** (feature/<feature name>, fix/<fix description>, etc.)
- **NEVER work directly on the main branch**

### At the End of Work

1. Create commit

### Commit Messages

- Use Japanese for commit messages (日本語でコミットメッセージを書く)
- Keep title under 50 characters (タイトルは50文字以内)
- Include purpose and expected impact (変更の目的と期待される影響を含める)
- Use natural line breaks, not `\n`
- Include main changes and feature additions when applicable:
  - **主な変更点：**
  - **機能追加**

### Pull Requests

- Only create PRs when explicitly requested (指示があるまでPRは作成しない)
- Use `gh` command for PR creation
- Include these sections in PR description:
  - **概要：** (Overview) - purpose and background
  - **変更内容：** (Changes) - specific modifications
  - **影響範囲：** (Impact) - effects on other parts
  - **テスト結果：** (Test Results) - testing status
- Add `Closes #**` if branch name contains `issue_**`
- Use markdown formatting where possible
