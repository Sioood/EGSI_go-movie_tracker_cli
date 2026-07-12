.PHONY: build test test-race cover fmt fmt-check vet lint deadcode check run-cli run-server tidy docker-up docker-down
.PHONY: build-linux build-windows build-darwin release-snapshot

VERSION ?= $(shell grep 'Version =' internal/version/version.go | cut -d'"' -f2)
LDFLAGS := -s -w -X github.com/movietracker/movie-tracker/internal/version.Version=$(VERSION)
GOLANGCI_LINT_VERSION := v1.64.8
DEADCODE_VERSION := latest

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

test-race:
	go test -race ./...

cover:
	go test -coverprofile=coverage.out ./...
	go tool cover -func=coverage.out | tail -1

fmt:
	@command -v goimports >/dev/null 2>&1 || go install golang.org/x/tools/cmd/goimports@latest
	gofmt -w .
	@PATH="$$(go env GOPATH)/bin:$$PATH" goimports -w .

fmt-check:
	@files="$$(gofmt -l .)"; if [ -n "$$files" ]; then echo "Fichiers non formatés :"; echo "$$files"; exit 1; fi

vet:
	go vet ./...

lint:
	@command -v golangci-lint >/dev/null 2>&1 || go install github.com/golangci/golangci-lint/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION)
	@PATH="$$(go env GOPATH)/bin:$$PATH" golangci-lint run ./...

deadcode:
	@command -v deadcode >/dev/null 2>&1 || go install golang.org/x/tools/cmd/deadcode@$(DEADCODE_VERSION)
	@PATH="$$(go env GOPATH)/bin:$$PATH" deadcode -test ./...

check: fmt-check vet test lint deadcode

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
