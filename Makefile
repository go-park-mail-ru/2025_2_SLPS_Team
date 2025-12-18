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
	swag init -g cmd/main/main.go
	docker-compose down -v
	docker-compose up -d postgres redis

	$(MAKE) wait-db
	migrate -path ./db/migrations -database "postgres://postgres:mysecretpassword@localhost:5432/vk?sslmode=disable" up

	go run ./cmd/app/main.go
reload-swagger:
	swag init -g cmd/main/main.godocker compose --env-file .env up --build


generate-mocks:
	mockgen -source=domain/auth.go -destination=internal/service/mocks/mock_auth.go -package=mocks
	mockgen -source=domain/chat.go -destination=internal/service/mocks/mock_chat.go -package=mocks
	mockgen -source=domain/profile.go -destination=internal/service/mocks/mock_profile.go -package=mocks
	mockgen -source=domain/post.go -destination=internal/service/mocks/mock_post.go -package=mocks
	mockgen -source=domain/friend.go -destination=internal/service/mocks/mock_friend.go -package=mocks
	mockgen -source=domain/community.go -destination=internal/service/mocks/mock_community.go -package=mocks CommunityStore
	mockgen -source=domain/comment.go -destination=internal/service/mocks/mock_comment.go -package=mocks

	# gRPC clients (для использования в сервисах)
	mockgen -source=shared/pb/auth_grpc.pb.go -destination=internal/service/mocks/mock_auth_grpc.go -package=mocks
	mockgen -source=shared/pb/profile_grpc.pb.go -destination=internal/service/mocks/mock_profile_grpc.go -package=mocks ProfileServiceClient
	mockgen -source=shared/pb/friend_grpc.pb.go -destination=internal/service/mocks/mock_friend_grpc.go -package=mocks FriendServiceClient
	
	# gRPC server interfaces (для тестирования хендлеров)
	mockgen -source=shared/pb/profile_grpc.pb.go -destination=internal/handler/grpc/mocks/mock_profile_server.go -package=mocks ProfileServiceServer
	mockgen -source=shared/pb/friend_grpc.pb.go -destination=internal/handler/grpc/mocks/mock_friend_server.go -package=mocks FriendServiceServer
	
	# Store interfaces
	mockgen -source=domain/profile.go -destination=internal/repository/mocks/mock_profile.go -package=mocks
	mockgen -source=domain/chat.go -destination=internal/repository/mocks/mock_chat.go -package=mocks
	mockgen -source=domain/message.go -destination=internal/repository/mocks/mock_message.go -package=mocks
	mockgen -source=domain/session.go -destination=internal/repository/mocks/mock_session.go -package=mocks
	mockgen -source=domain/user.go -destination=internal/repository/mocks/mock_user.go -package=mocks
	mockgen -source=domain/post.go -destination=internal/repository/mocks/mock_post.go -package=mocks
	mockgen -source=domain/friend.go -destination=internal/repository/mocks/mock_friend.go -package=mocks
	mockgen -source=domain/friend.go -destination=internal/repository/mocks/mock_friend.go -package=mocks
	mockgen -source=domain/comment.go -destination=internal/repository/mocks/mock_comment.go -package=mocks
test-coverage:
	@rm -f coverage.out coverage_filtered.out
	go clean -testcache
	go test -v -coverprofile=coverage.out -coverpkg=./... ./...
	grep -v -E "(docs|fill\.go|mock.*\.go|generate\.go|test_utils\.go|\.pb\.go|.*_easyjson\.go|.*_gen\.go)" coverage.out > coverage_filtered.out || true
	go tool cover -func=coverage_filtered.out | grep total
test-coverage-html: test-coverage
	go tool cover -html=coverage_filtered.out -o coverage.html

generate-easyjson:
	easyjson -all -pkg ./domain/



