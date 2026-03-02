.PHONY: test test-race test-cover lint vet build clean

# Run all tests
test:
	go test ./... -count=1

# Run tests with race detector
test-race:
	go test -race ./... -count=1

# Run tests with coverage report
test-cover:
	go test -coverprofile=coverage.out ./...
	go tool cover -func=coverage.out
	@echo "---"
	@echo "To view HTML report: go tool cover -html=coverage.out"

# Run go vet
vet:
	go vet ./...

# Run staticcheck (install: go install honnef.co/go/tools/cmd/staticcheck@latest)
lint: vet
	@which staticcheck > /dev/null 2>&1 && staticcheck ./... || echo "staticcheck not installed, skipping"

# Build all packages
build:
	go build ./...

# Run all checks (CI equivalent)
ci: vet build test-race test-cover

# Clean build artifacts
clean:
	rm -f coverage.out
	rm -f /track17_bin
