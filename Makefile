.PHONY: build test lint run-cli run-server tidy docker-up docker-down
.PHONY: build-linux build-windows build-darwin release-snapshot

VERSION ?= $(shell grep 'Version =' internal/version/version.go | cut -d'"' -f2)
LDFLAGS := -s -w -X github.com/movietracker/movie-tracker/internal/version.Version=$(VERSION)

build:
	go build -ldflags "$(LDFLAGS)" -o bin/movietracker ./cmd/cli
	go build -ldflags "$(LDFLAGS)" -o bin/movietracker-server ./cmd/server

build-linux:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o bin/movietracker-linux-amd64 ./cmd/cli
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o bin/movietracker-server-linux-amd64 ./cmd/server

build-windows:
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o bin/movietracker-windows-amd64.exe ./cmd/cli

build-darwin:
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o bin/movietracker-darwin-amd64 ./cmd/cli
	CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -ldflags "$(LDFLAGS)" -o bin/movietracker-darwin-arm64 ./cmd/cli

release-snapshot:
	goreleaser release --snapshot --clean

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
