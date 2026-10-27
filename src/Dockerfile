# Сборка: версия Go не ниже строки `go` в go.mod.
FROM golang:1.27-alpine AS build
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
# Статический бинарник без C-зависимостей; часовые пояса встроены (time/tzdata в main.go).
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /server ./cmd/server

FROM alpine:3.22
RUN adduser -D -H zapas
COPY --from=build /server /server
USER zapas
EXPOSE 8080
HEALTHCHECK --interval=5s --timeout=3s --retries=10 CMD wget -qO- http://localhost:8080/health || exit 1
ENTRYPOINT ["/server"]
