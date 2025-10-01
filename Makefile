.PHONY: test-coverage

test-coverage:
	go test -covermode=count -coverpkg=./... -coverprofile=coverage.out ./internal/handler
	go tool cover -func=coverage.out
	go tool cover -html=coverage.out
