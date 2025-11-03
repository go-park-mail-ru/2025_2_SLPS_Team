.PHONY: test-coverage

test-coverage:
	go test -covermode=count -coverpkg=./... -coverprofile=coverage.out ./internal/handler
	go tool cover -func=coverage.out
	go tool cover -html=coverage.out
migrations-up:
	migrate -path ./db/migrations -database "postgres://postgres:mysecretpassword@localhost:5432/vk?sslmode=disable" up
migrations-down:
	migrate -path ./db/migrations -database "postgres://postgres:mysecretpassword@localhost:5432/vk?sslmode=disable" down
wait-db:
	powershell -Command "while (-not (Test-NetConnection -ComputerName localhost -Port 5432).TcpTestSucceeded) { Start-Sleep -Seconds 1 }"

reload-db:
	swag init -g cmd/app/main.go
	docker-compose down -v
	docker-compose up -d postgres redis

	$(MAKE) wait-db
	migrate -path ./db/migrations -database "postgres://postgres:mysecretpassword@localhost:5432/vk?sslmode=disable" up

	go run ./cmd/app/main.go
reload-swagger:
	swag init -g cmd/app/main.go
