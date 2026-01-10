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

- `/`: Root package code
  - `knownbots.go` - Utility functions and core types
  - `validator.go` - Validator logic and methods
  - `bot.go` - Bot type and loading
  - `config.go` - Config struct and Option functions
  - `embed.go` - Embedded bot configurations (go:embed)
  - `ua.go` - User-Agent classification
  - `cache.go` - RDNS cache implementation
  - `lru.go` - LRU cache for failed lookups
- `bots/`: Configuration for known bots (embedded in binary)
  - `conf.d/`: YAML configuration files for individual bots (57 built-in bots)
- `parser/`: IP range parser implementations.
  - `parser.go` - Parser interface and registry
  - `parser_google.go` - Google-style JSON parser (ipv4Prefix/ipv6Prefix)
  - `parser_openai.go` - OpenAI-style JSON parser (prefix field)
  - `parser_stripe.go` - Stripe webhook IP parser
  - `parser_github.go` - GitHub API IP parser
  - `parser_txt.go` - Plain text line-by-line parser

## Bot Configuration

Bot definitions are stored in YAML files within `bots/conf.d/` and embedded in the binary.

**Built-in Bots**: The library includes 57 built-in bot configurations (Googlebot, Bingbot, GPTBot, etc.) that are embedded via `go:embed` and loaded automatically.

**Custom Bots**: Users can add custom bots by placing YAML files in their own `./bots/conf.d/` directory. Custom bots override built-in bots with the same name.

Each bot configuration file should define:

```yaml
kind: SearchEngine        # Bot category (SearchEngine, SocialMedia, AITraining, AIAssist, etc.)
name: googlebot           # Unique identifier for the bot
parser: google            # Parser name: google, openai, txt, github, stripe
ua: "Googlebot"           # User-Agent string fragment (case-sensitive)
urls:                     # List of URLs to fetch IP lists (auto-downloaded)
  - "https://www.gstatic.com/ipranges/google.json"
custom: []                # Static CIDR ranges (optional, for backup IPs)
domains: []               # Verified RDNS domains (only if rdns: true)
rdns: false               # Enable RDNS verification (default: false, use URLs instead)
```

### Configuration Loading

The `Load()` function:
1. Loads all 57 built-in bots from embedded configuration
2. Loads custom bots from `./bots/conf.d/` (if directory exists)
3. Custom bots override built-in bots with the same name (with warning log)

### Available Parsers

| Parser | JSON Structure | Example Bots |
|--------|---------------|--------------|
| `google` | `{prefixes: [{ipv4Prefix, ipv6Prefix}]}` | Googlebot, Bingbot |
| `openai` | `{prefixes: [{prefix}]}` | GPTBot, ClaudeBot |
| `txt` | Plain text, one CIDR per line | UptimeRobot |
| `github` | `{hooks, importer, web}` | GitHub webhooks |
| `stripe` | `{webhooks: {ipv4, ipv6}}` | Stripe webhooks |

**Key Principles**:
- Use `urls` + `rdns: false` (or omit rdns) for bots with official JSON IP lists
- Use `domains` + `rdns: true` only for bots without official IP lists (Baidu, Yandex, etc.)
- Empty fields (`custom: []`, `rdns: false`) should be omitted (default values)

## Cursor & Copilot Rules

*(No specific .cursor/rules or .github/copilot-instructions.md were found in the repository. Adhere to the standard Go conventions outlined above.)*