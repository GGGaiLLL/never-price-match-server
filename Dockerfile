# 1. 构建阶段
FROM golang:1.22-alpine AS builder
ENV GOTOOLCHAIN=auto
WORKDIR /app
RUN apk add --no-cache git
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o server ./cmd/server

# 2. 运行阶段
FROM gcr.io/distroless/base-debian12
WORKDIR /app
COPY --from=builder /app/server /app/server
COPY configs /app/configs
EXPOSE 8080
USER 65532:65532
ENTRYPOINT ["/app/server"]
