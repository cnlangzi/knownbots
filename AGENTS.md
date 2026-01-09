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

### 6. Testing
- Place tests in `_test.go` files alongside the code they test.
- Use the standard `testing` package.
- Use `t.TempDir()` for creating temporary directories in tests (as seen in `TestLoadConfigs`).
- Table-driven tests are encouraged for logic validation (as seen in `TestContainsIP`).

## Directory Structure

- `/`: Root package code (`knownbots.go`, `validator.go`).
- `bots/`: Configuration for known bots.
  - `conf.d/`: YAML configuration files for individual bots.
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


- 当需要搜索文档时，使用 `context7` 工具。