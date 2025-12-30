.PHONY: build run clean install

# Build the application
build:
	go build -o bin/desktop-app ./cmd/picopeeker

# Run the application
run:
	go run ./cmd/picopeeker

# Clean build artifacts
clean:
	rm -rf bin/

# Install dependencies
install:
	go mod download
	go mod tidy

# Build for release (optimized)
release:
	mkdir -p bin
	go build -ldflags="-s -w" -o bin/desktop-app ./cmd/picopeeker

# Run tests (if you add tests later)
test:
	go test ./...
