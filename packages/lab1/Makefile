# HTTP File Server Makefile

.PHONY: help dev server client stop build clean test logs status downloads

help:
	@echo "HTTP File Server - Available Commands:"
	@echo "  dev        - Start development server with file watching"
	@echo "  server     - Start HTTP server in background"
	@echo "  client     - Run client (set HOST, PORT, FILE, DIR)"
	@echo "  stop       - Stop all services"
	@echo "  build      - Build Docker images"
	@echo "  clean      - Clean all containers, images, and downloads"
	@echo "  test       - Run full test suite"
	@echo "  logs       - Show server logs"
	@echo "  status     - Show container status"
	@echo "  downloads  - List downloaded files"

dev:
	docker-compose watch

server:
	docker-compose up -d http-server

client:
	$(eval HOST := $(if $(HOST),$(HOST),http-server))
	$(eval PORT := $(if $(PORT),$(PORT),8080))
	$(eval FILE := $(if $(FILE),$(FILE),index.html))
	$(eval DIR := $(if $(DIR),$(DIR),./downloads))
	SERVER_HOST=$(HOST) SERVER_PORT=$(PORT) URL_PATH=$(FILE) DOWNLOAD_DIR=$(DIR) \
	docker-compose --profile client run --rm http-client

# Stop the server
stop:
	docker-compose down

build:
	docker-compose build

clean:
	docker-compose down --rmi all --volumes --remove-orphans
	@if [ -d "downloads" ]; then \
		rm -rf downloads/*; \
		echo "Downloads directory cleared"; \
	fi

logs:
	docker-compose logs -f http-server

test: clean server
	@echo "Test 1: Download HTML file"
	@make client FILE=index.html
	@echo ""
	@echo "Test 2: Download PDF file"
	@make client FILE=assets/1962-calhoun.pdf
	@echo ""
	@echo "Test 3: Browse assets directory"
	@make client FILE=assets/
	@echo ""
	@echo "Test 4: Test 404 error"
	@make client FILE=nonexistent.html || true
	@echo ""
	@echo "Tests completed. Stopping server..."
	@make stop

status:
	@docker-compose ps

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
