.PHONY: test-coverage

test-coverage:
	go test -covermode=count -coverpkg=./... -coverprofile=coverage.out ./internal/handler
	go tool cover -func=coverage.out
	go tool cover -html=coverage.out
migrations-up:
	migrate -path ./repository/migrations -database "postgres://postgres:mysecretpassword@localhost:5432/vk?sslmode=disable" up
migrations-down:
	migrate -path ./repository/migrations -database "postgres://postgres:mysecretpassword@localhost:5432/vk?sslmode=disable" down
wait-db:
	powershell -Command "while (-not (Test-NetConnection -ComputerName localhost -Port 5432).TcpTestSucceeded) { Start-Sleep -Seconds 1 }"

reload-db:
	docker-compose down -v
	docker-compose up -d

	$(MAKE) wait-db
	migrate -path ./repository/migrations -database "postgres://postgres:mysecretpassword@localhost:5432/vk?sslmode=disable" up

	go run ./cmd/main.go

