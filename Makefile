.PHONY: all
all: test build docker-build

.PHONY: build
build:
	# go test ./...
	mkdir -p dist
	go generate ./...
	go build -o ./dist/router ./cmd/router
	go build -o ./dist/worker ./cmd/worker

.PHONY: test
test:
	go test -v ./...

.PHONY: docker-build
docker-build:
	docker build -t fair-router .

