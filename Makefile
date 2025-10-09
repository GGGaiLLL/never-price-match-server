APP=server

.PHONY: run build fmt vet docker

run:
	APP_ENV=local go run ./cmd/server

build:
	go build -o bin/$(APP) ./cmd/server

fmt:
	go fmt ./...

vet:
	go vet ./...

docker:
	docker build -t price-match:latest .
