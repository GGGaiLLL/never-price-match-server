APP=server

.PHONY: run build fmt vet docker

mod:
	go mod tidy

init:
	go run github.com/99designs/gqlgen init

gen:
	go run github.com/99designs/gqlgen@v0.17.81 generate --config tools/gqlgen.yml

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
