.PHONY: build test run simulation clean help format dev-test

# Default target
help:
	@echo "Memory Scramble - Lab 3"
	@echo ""
	@echo "Available targets:"
	@echo "  build      - Build the server binary"
	@echo "  test       - Run unit tests"
	@echo "  run        - Start the game server (default port 8080)"
	@echo "  simulation - Run the simulation script"
	@echo "  format     - Format Go source code"
	@echo "  clean      - Clean build artifacts"
	@echo "  help       - Show this help"
	@echo ""
	@echo "Usage examples:"
	@echo "  make run                    # Start server on port 8080"
	@echo "  make run PORT=9000          # Start server on port 9000"
	@echo "  make run BOARD=boards/custom.txt  # Use custom board file"
	@echo ""
	@echo "Documentation:"
	@echo "  See docs/ directory for comprehensive API documentation"
	@echo "  Use 'go doc ./internal/board' to view Go package documentation"

# Build the server
build:
	go build -o memory-scramble main.go

# Run tests
test:
	go test ./test/...

# Run the server (with optional PORT and BOARD parameters)
PORT ?= 8080
BOARD ?= boards/perfect.txt
run:
	@echo "Starting Memory Scramble server..."
	@echo "Port: $(PORT)"
	@echo "Board: $(BOARD)"
	@echo "Open http://localhost:$(PORT) in your browser to play!"
	go run main.go -port=$(PORT) -board=$(BOARD)

# Run simulation
simulation:
	@echo "Running Memory Scramble simulation..."
	go run cmd/simulation/main.go

# Clean build artifacts
clean:
	rm -f memory-scramble
	go clean

# Development helpers
dev-test:
	go test ./test/... -v

format:
	go fmt ./...

