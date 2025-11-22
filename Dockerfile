# Этап 1: Сборка приложения
FROM golang:1.25.1 AS builder

WORKDIR /app

# Копируем только файлы, необходимые для загрузки зависимостей
COPY go.mod go.sum ./
RUN go mod download

# Копируем остальные файлы
COPY . .

# Собираем оба приложения
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/app ./cmd/app
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/migration ./cmd/migration

# Этап 2: Финальный образ
FROM alpine:3.18

WORKDIR /app

# Устанавливаем зависимости для миграций
RUN apk add --no-cache postgresql-client

# Копируем только необходимые артефакты
COPY --from=builder /out/app ./main
COPY --from=builder /out/migration ./migrate
COPY --from=builder /app/db/migrations ./db/migrations
COPY --from=builder /app/logs ./logs
EXPOSE 8080

# Запускаем миграции и приложение
CMD sh -c "./migrate && ./main"
