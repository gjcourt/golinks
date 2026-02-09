.PHONY: build run test clean dev

# Build the application
build:
	go build -o golinks ./cmd/golinks

# Run the application
run: build
	./golinks

# Run tests
test:
	go test -race -v ./...

# Clean build artifacts
clean:
	rm -f golinks
	rm -f golinks.db

# Development mode with hot reload (requires air)
dev:
	@which air > /dev/null || (echo "Installing air..." && go install github.com/cosmtrek/air@latest)
	air

# Download dependencies
deps:
	go mod download
	go mod tidy

# Build for multiple platforms
build-all:
	GOOS=linux GOARCH=amd64 go build -o golinks-linux-amd64 ./cmd/golinks
	GOOS=darwin GOARCH=amd64 go build -o golinks-darwin-amd64 ./cmd/golinks
	GOOS=darwin GOARCH=arm64 go build -o golinks-darwin-arm64 ./cmd/golinks
	GOOS=windows GOARCH=amd64 go build -o golinks-windows-amd64.exe ./cmd/golinks
