.PHONY: build run clean install

# Build the application
build:
	go build -o bin/desktop-app main.go

# Run the application
run:
	go run main.go

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
	go build -ldflags="-s -w" -o bin/desktop-app main.go

# Run tests (if you add tests later)
test:
	go test ./...
