# KnownBots Agent Guidelines

This document provides instructions for AI agents (and human developers) working on the KnownBots project.

## Project Overview

**knownbots** is a Go library for verifying search engine crawlers and identifying legitimate bots. It validates bots based on User-Agent strings and IP addresses.

## Development Environment & Commands

### 1. Build & Test
This project uses standard Go tooling.

- **Run all tests:**
  ```bash
  go test ./...
  ```

- **Run a single test:**
  ```bash
  go test -v -run ^TestName$ ./...
  # Example:
  go test -v -run ^TestValidator$ ./...
  ```

- **Run tests with coverage:**
  ```bash
  go test -cover ./...
  ```

- **Update dependencies:**
  ```bash
  go mod tidy
  ```

### 2. Linting
This project does not currently have a dedicated linter config file (like `.golangci.yml`), but code should be formatted using standard Go tools.

- **Format code:**
  ```bash
  go fmt ./...
  ```

- **Vet code (standard Go static analysis):**
  ```bash
  go vet ./...
  ```

## Code Style & Conventions

Follow idiomatic Go (Golang) practices.

### 1. Formatting
- Always use `gofmt` (or `go fmt`) to format your code.
- Indentation uses tabs, not spaces.

### 2. Naming
- **File Names:** Use snake_case (e.g., `knownbots_test.go`, `validator.go`).
- **Function/Method Names:** Use PascalCase for exported functions (e.g., `LoadConfigs`) and camelCase for private ones (e.g., `containsIP`).
- **Variable Names:** Short and descriptive.
  - Good: `v` for Validator, `err` for error, `ip` for IP address.
  - Bad: `validatorInstance`, `theError`, `ipAddressString`.

### 3. Imports
- Group imports into standard library, third-party packages, and internal packages.
- Use `go mod tidy` to manage imports in `go.mod`.

### 4. Error Handling
- Return errors as the last return value.
- Use `fmt.Errorf` with `%w` to wrap errors when adding context.
- Check errors immediately after the function call.
  ```go
  if err := doSomething(); err != nil {
      return fmt.Errorf("failed to do something: %w", err)
  }
  ```

### 5. Types & Interfaces
- Define types and interfaces where they make sense for abstraction.
- Keep structs small and focused.

### 6. File Organization (CRITICAL)
**Split files by responsibility - do not mix unrelated code in the same file.**

Each file should have a single, focused responsibility:
- `parser.go` - Parser interface and registry
- `parser_*.go` - Individual parser implementations (e.g., `parser_google.go`, `parser_txt.go`)
- `validator.go` - Validator logic
- `bot.go` - Bot type and loading
- `cache.go` - Cache implementation
- `lru.go` - LRU cache

If a file grows beyond ~150-200 lines, consider splitting it by responsibility.

### 7. Testing
- Place tests in `_test.go` files alongside the code they test.
- Use the standard `testing` package.
- Use `t.TempDir()` for creating temporary directories in tests (as seen in `TestLoadConfigs`).
- Table-driven tests are encouraged for logic validation (as seen in `TestContainsIP`).

## API Documentation Tools

### Context7 MCP
**Use Context7 MCP for Go standard library and package documentation.**

When you need to look up Go API documentation:
1. Use the Context7 MCP tool (not web search) for official Go docs
2. Examples:
   - Look up `net/netip` package APIs
   - Check `sync/atomic` usage patterns
   - Verify `io.Reader` interface contracts

This ensures accurate, up-to-date information from official sources.

## Directory Structure

- `/`: Root package code (`knownbots.go`, `validator.go`).
- `bots/`: Configuration for known bots.
  - `conf.d/`: YAML configuration files for individual bots.
- `parser/`: IP range parser implementations.
  - `parser.go` - Parser interface and registry
  - `parser_*.go` - Individual parser implementations
- `tasks/`: Task definitions or documentation.

## Bot Configuration

Bot definitions are stored in YAML files within `bots/conf.d/`.
Each file should define:
- `name`: Unique identifier for the bot.
- `ua`: User-Agent string fragment to match.
- `ips`: List of CIDR ranges for the bot's IPs.
- `domains`: List of verified domains.
- `urls`: List of URLs to fetch IP lists (optional).
- `extra`: Additional IP ranges (optional).

## Cursor & Copilot Rules

*(No specific .cursor/rules or .github/copilot-instructions.md were found in the repository. Adhere to the standard Go conventions outlined above.)*