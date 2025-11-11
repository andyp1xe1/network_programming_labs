.PHONY: build test run simulation clean help docs docs-serve format dev-test

# Default target
help:
	@echo "Memory Scramble - Lab 3"
	@echo ""
	@echo "Available targets:"
	@echo "  build      - Build the server binary"
	@echo "  test       - Run unit tests"
	@echo "  run        - Start the game server (default port 8080)"
	@echo "  simulation - Run the simulation script"
	@echo "  docs       - Generate Go documentation files"
	@echo "  docs-serve - Start godoc server for interactive documentation"
	@echo "  format     - Format Go source code"
	@echo "  clean      - Clean build artifacts"
	@echo "  help       - Show this help"
	@echo ""
	@echo "Usage examples:"
	@echo "  make run                    # Start server on port 8080"
	@echo "  make run PORT=9000          # Start server on port 9000"
	@echo "  make run BOARD=boards/custom.txt  # Use custom board file"
	@echo "  make docs                   # Generate documentation files"
	@echo "  make docs-serve             # Browse documentation at localhost:6060"

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

# Documentation generation
docs:
	@echo "Generating Go documentation..."
	@mkdir -p docs
	@go doc -all > docs/API_DOCUMENTATION.md
	@echo "# Lab 3 Go Package Documentation\n" > docs/GODOC.md
	@echo "Generated on: $$(date)\n" >> docs/GODOC.md
	@echo "## Package: main\n" >> docs/GODOC.md
	@go doc . >> docs/GODOC.md
	@echo "\n## Package: board\n" >> docs/GODOC.md
	@go doc -all ./internal/board >> docs/GODOC.md
	@echo "\n## Package: commands\n" >> docs/GODOC.md
	@go doc -all ./internal/commands >> docs/GODOC.md
	@echo "\n## Package: server\n" >> docs/GODOC.md
	@go doc -all ./internal/server >> docs/GODOC.md
	@echo "Documentation generated in docs/"

# Start godoc server for local browsing
docs-serve:
	@echo "Starting godoc server on http://localhost:6060"
	@echo "Navigate to http://localhost:6060/pkg/lab3/ to view documentation"
	godoc -http=:6060