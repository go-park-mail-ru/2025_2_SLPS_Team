package grpcclient

import (
	"context"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func WaitForGRPC[T any](target string, constructor func(grpc.ClientConnInterface) T, retries int, delay time.Duration) (T, *grpc.ClientConn, error) {
	var zero T
	var lastErr error

	for i := 0; i < retries; i++ {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		conn, err := grpc.DialContext(ctx, target, grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithBlock())
		if err == nil {
			client := constructor(conn)
			return client, conn, nil
		}
		lastErr = err
		time.Sleep(delay)
	}
	return zero, nil, lastErr
}
