.PHONY: test-coverage

#test-coverage:
#	go test -covermode=count -coverpkg=./... -coverprofile=coverage.out ./internal/handler
#	go tool cover -func=coverage.out
#	go tool cover -html=coverage.out
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


generate-mocks:
	mockgen -source=domain/auth.go -destination=internal/service/mocks/mock_auth.go -package=mocks AuthService
	mockgen -source=domain/chat.go -destination=internal/service/mocks/mock_chat.go -package=mocks ChatService
	mockgen -source=domain/profile.go -destination=internal/service/mocks/mock_profile.go -package=mocks ProfileService
	
	mockgen -source=domain/post.go -destination=internal/service/mocks/mock_post.go -package=mocks PostService
	mockgen -source=domain/friend.go -destination=internal/service/mocks/mock_friend.go -package=mocks FriendService
	
	mockgen -source=domain/profile.go -destination=internal/repository/mocks/mock_profile.go -package=mocks ProfileStore
	mockgen -source=domain/chat.go -destination=internal/repository/mocks/mock_chat.go -package=mocks ChatStore
	mockgen -source=domain/message.go -destination=internal/repository/mocks/mock_message.go -package=mocks MessageStore
	mockgen -source=domain/session.go -destination=internal/repository/mocks/mock_session.go -package=mocks SessionStore
	mockgen -source=domain/user.go -destination=internal/repository/mocks/mock_user.go -package=mocks UserStore

	mockgen -source=domain/post.go -destination=internal/repository/mocks/mock_post.go -package=mocks PostStore
	mockgen -source=domain/friend.go -destination=internal/repository/mocks/mock_friend.go -package=mocks FriendStore
	
test-coverage:
	@rm -f coverage.out coverage_filtered.out
	go clean -testcache
	go test -v -coverprofile=coverage.out -coverpkg=./... ./...
	grep -v -E "(docs|fill\.go|mock.*\.go|generate\.go)" coverage.out > coverage_filtered.out || true
	go tool cover -func=coverage_filtered.out | grep total

test-coverage-html: test-coverage
	go tool cover -html=coverage_filtered.out -o coverage.html
