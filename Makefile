# Makefile for OpenAI and Anthropic to OpenRouter API Router

# Variables
BINARY_NAME=router
DOCKER_IMAGE_NAME=openai-anthropic-openrouter-router
GO_FILES=$(wildcard *.go)

# Go commands
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get

# Docker commands
DOCKER=docker
DOCKER_COMPOSE=docker-compose

.PHONY: all build clean test run docker-build docker-run docker-compose-up docker-compose-down

all: test build

build:
	$(GOBUILD) -o $(BINARY_NAME) -v

clean:
	$(GOCLEAN)
	rm -f $(BINARY_NAME)

test:
	$(GOTEST) -v ./...

run: build
	./$(BINARY_NAME)

run-with-options: build
	./$(BINARY_NAME) --middleware middleware.js --system system_prompt.txt --router router.js

deps:
	$(GOGET) github.com/dop251/goja
	$(GOGET) github.com/joho/godotenv

docker-build:
	$(DOCKER) build -t $(DOCKER_IMAGE_NAME) .

docker-run: docker-build
	$(DOCKER) run -p 80:80 \
		-v $(PWD)/.env:/app/.env \
		-v $(PWD)/middleware.js:/app/middleware.js \
		-v $(PWD)/system_prompt.txt:/app/system_prompt.txt \
		-v $(PWD)/router.js:/app/router.js \
		$(DOCKER_IMAGE_NAME) \
		./main --nohosts --router /app/router.js

docker-compose-up:
	$(DOCKER_COMPOSE) up --build

docker-compose-down:
	$(DOCKER_COMPOSE) down

# Helper target to check and create necessary files
check-files:
	@test -f .env || cp .env.example .env
	@test -f middleware.js || echo "function process(request) { return request; }" > middleware.js
	@test -f system_prompt.txt || echo "You are a helpful assistant." > system_prompt.txt
	@test -f router.js || echo "function route(request) { return { model: process.env.OR_MODEL, url: process.env.OR_ENDPOINT, bearer: process.env.OR_KEY }; }" > router.js

# Target to set up the project
setup: deps check-files

# Target to run the project with all options
run-full: check-files run-with-options