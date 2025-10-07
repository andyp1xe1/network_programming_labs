# HTTP File Server Makefile

.PHONY: help server client stop build clean test logs status downloads

# Default target
help:
	@echo "HTTP File Server - Available commands:"
	@echo ""
	@echo "  make dev             - Start development mode via docker-compose watch"
	@echo "  make server          - Start the HTTP server (docker-compose up)"
	@echo "  make client FILE=<filename> [HOST=<address>] [PORT=<port>] - Run client to download file"
	@echo "  make stop            - Stop the server"
	@echo "  make build           - Build Docker images"
	@echo "  make clean           - Stop and remove containers/images/downloads"
	@echo "  make test            - Run basic tests"
	@echo "  make logs            - Show server logs"
	@echo "  make status          - Check if server is running"
	@echo "  make downloads       - Show downloaded files tree"
	@echo ""
	@echo "Examples:"
	@echo "  make server"
	@echo "  make client FILE=index.html"
	@echo "  make client FILE=index.html HOST=192.168.1.100"
	@echo "  make client FILE=index.html HOST=192.168.1.100 PORT=9000"

dev:
	@echo "Watching for changes..."
	docker-compose watch

# Start the HTTP server
server:
	@echo "Starting HTTP file server..."
	docker-compose up -d http-server

# Run client with specified file
client:
	@if [ -z "$(FILE)" ]; then \
		echo "Error: Please specify FILE parameter"; \
		echo "Usage: make client FILE=<filename> [HOST=<address>] [PORT=<port>]"; \
		echo "Example: make client FILE=index.html"; \
		echo "Example: make client FILE=index.html HOST=192.168.1.100"; \
		exit 1; \
	fi
	$(eval HOST := $(if $(HOST),$(HOST),http-server))
	$(eval PORT := $(if $(PORT),$(PORT),8080))
	@echo "Downloading: $(FILE) from $(HOST):$(PORT)"
	docker-compose run --rm http-client \
		uv run ./packages/lab1/src/lab1/client.py $(HOST) $(PORT) $(FILE)

# Stop the server
stop:
	@echo "Stopping HTTP file server..."
	docker-compose down

# Build Docker images
build:
	@echo "Building Docker images..."
	docker-compose build

# Clean up containers, images, and downloads
clean:
	@echo "Cleaning up containers, images, and downloads..."
	docker-compose down --rmi all --volumes --remove-orphans
	@if [ -d "downloads" ]; then \
		rm -rf downloads/*; \
		echo "Downloads directory cleared"; \
	fi

# Show server logs
logs:
	docker-compose logs -f http-server

# Run basic tests
test: clean server
	@echo "Running basic tests..."
	@sleep 3
	@echo "Test 1: Download HTML file"
	@make client FILE=index.html
	@echo ""
	@echo "Test 2: Download PDF file"
	@make client FILE=document1.pdf
	@echo ""
	@echo "Test 3: Browse directory"
	@make client FILE=subdir/
	@echo ""
	@echo "Test 4: Test 404 error"
	@make client FILE=nonexistent.html || true
	@echo ""
	@echo "Tests completed. Stopping server..."
	@make stop

# Check if server is running
status:
	@docker-compose ps

# View downloaded files using tree
downloads:
	@echo "Downloaded files:"
	@if [ -d "downloads" ]; then \
		if command -v tree >/dev/null 2>&1; then \
			tree downloads/; \
		else \
			ls -la downloads/; \
		fi \
	else \
		echo "No downloads directory found"; \
	fi
