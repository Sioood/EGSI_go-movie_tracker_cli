.PHONY: build test run-cli run-server tidy

build:
	go build -o bin/movietracker ./cmd/cli
	go build -o bin/movietracker-server ./cmd/server

test:
	go test ./...

run-cli: build
	./bin/movietracker

run-server: build
	./bin/movietracker-server

tidy:
	go mod tidy
