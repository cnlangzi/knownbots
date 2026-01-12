.PHONY: test test-short test-integration bench fmt vet all

# Go commands
GOCMD = go
GOFMT = $(GOCMD) fmt
GOVET = $(GOCMD) vet
GOTEST = $(GOCMD) test
GOBENCH = $(GOCMD) test -run=^$

# Default target
all: fmt vet test bench

# Format code
fmt:
	@echo "Formatting code..."
	$(GOFMT) ./...

# Vet code
vet:
	@echo "Vetting code..."
	$(GOVET) ./...

# Run unit tests (short mode, no integration tests)
test-short:
	@echo "Running unit tests..."
	$(GOTEST) -short ./... -race -coverprofile=coverage.txt -covermode=atomic

# Run all tests including integration tests
test:
	@echo "Running all tests..."
	$(GOTEST) ./... -race -coverprofile=coverage.txt -covermode=atomic

# Run integration tests only
test-integration:
	@echo "Running integration tests..."
	$(GOTEST) ./asn/... -v -run "TestIntegration_"

# Run benchmarks
bench:
	@echo "Running benchmarks..."
	$(GOBENCH) -bench=. -benchmem -cpu=1,4,8 -timeout 120s

# Run benchmarks and output to file
bench-output:
	@echo "Running benchmarks..."
	$(GOBENCH) -bench=. -benchmem -cpu=1,4,8 -timeout 120s > benchmark_output.txt
