# HTTP File Server Makefile

.PHONY: help dev server client stop build clean test logs status downloads

dev:
	docker-compose watch

server:
	docker-compose up -d http-server

client:
	$(eval HOST := $(if $(HOST),$(HOST),http-server))
	$(eval PORT := $(if $(PORT),$(PORT),8080))
	docker-compose --profile client run --rm http-client \
		uv run ./packages/lab1/src/lab1/client.py $(HOST) $(PORT) $(FILE)

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
