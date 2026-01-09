SHELL := /bin/bash
COMPOSE := docker compose

.PHONY: up down logs ps test lint gen

up:
	$(COMPOSE) up -d --build

down:
	$(COMPOSE) down -v

logs:
	$(COMPOSE) logs -f --tail=200

ps:
	$(COMPOSE) ps

test:
	go test ./...

lint:
	golangci-lint run

gen:
	$(COMPOSE) run --rm buf generate
