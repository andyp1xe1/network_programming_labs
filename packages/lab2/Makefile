.PHONY: help dev server stop build clean logs status

dev:
	docker-compose watch

server:
	docker-compose up -d concurrent-server

stop:
	docker-compose down

build:
	docker-compose build

clean:
	docker-compose down --rmi all --volumes --remove-orphans

logs:
	docker-compose logs -f concurrent-server

status:
	@docker-compose ps
