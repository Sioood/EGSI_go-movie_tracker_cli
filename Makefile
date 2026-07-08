.PHONY: build test lint run-cli run-server tidy docker-up docker-down

build:
	go build -o bin/movietracker ./cmd/cli
	go build -o bin/movietracker-server ./cmd/server

test:
	go test ./...

lint:
	@command -v golangci-lint >/dev/null 2>&1 || go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	@PATH="$$(go env GOPATH)/bin:$$PATH" golangci-lint run ./...

run-cli: build
	./bin/movietracker

run-server: build
	./bin/movietracker-server

tidy:
	go mod tidy

docker-up:
	docker compose up --build -d

docker-down:
	docker compose down
