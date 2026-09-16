.DEFAULT_GOAL := help

COMPOSE ?= docker compose -f deployments/docker-compose.yaml
BACKEND ?= backend
FRONTEND ?= frontend
GO ?= go

.PHONY: help up down tidy migrate backend frontend

help:
	@echo "make up         启动 PostgreSQL 和 Redis，并 go mod tidy"
	@echo "make down       停止依赖容器"
	@echo "make tidy       整理 backend 的 Go 模块（go.sum）"
	@echo "make migrate    整理依赖后执行 migration up"
	@echo "make backend    启动 Gin（默认 :8080）"
	@echo "make frontend   启动 React（默认 :5173，/api 代理到后端）"

tidy:
	cd $(BACKEND) && $(GO) mod tidy

up: tidy
	$(COMPOSE) up -d postgres redis

down:
	$(COMPOSE) down

migrate: tidy
	cd $(BACKEND) && $(GO) run ./cmd/migrate -action up

backend: tidy
	cd $(BACKEND) && $(GO) run ./cmd/server -config application.yaml

frontend:
	cd $(FRONTEND) && pnpm install && pnpm dev
