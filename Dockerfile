# -----------------------
# Этап 1: Сборка приложения
# -----------------------
FROM golang:1.25.1 AS builder

WORKDIR /app

# Копируем только файлы для загрузки зависимостей
COPY go.mod go.sum ./
RUN go mod download

# Копируем остальные файлы
COPY . .
ARG SERVICE=main
# Аргумент для указания сервиса, который хотим собрать
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/$SERVICE ./cmd/$SERVICE

# -----------------------
# Этап 2: Финальный образ
# -----------------------
FROM alpine:3.18

WORKDIR /app

# Устанавливаем необходимые утилиты (например, для миграций)
RUN apk add --no-cache postgresql-client

# Копируем бинарь сервиса
ARG SERVICE=authservice
COPY --from=builder /out/$SERVICE ./main


COPY --from=builder /app/logs ./logs

# Порт gRPC сервиса
EXPOSE 8080

# Точка входа
CMD ["./main"]

