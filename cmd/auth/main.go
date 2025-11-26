package main

import (
	"log"
	"net"
	"project/cmd/dbconn"
	"project/cmd/grpcclient"
	"project/config"
	authHandler "project/internal/grpc"
	"project/internal/repository/db"
	"project/internal/repository/dbElastic"
	"project/internal/repository/dbRedis"
	"project/internal/service"
	"project/shared/pb"
	"time"

	"google.golang.org/grpc"
)

func main() {
	cfg := config.NewConfig()
	//loger := logger.NewLogger(cfg)

	dbConn := dbconn.NewPostgres(cfg.PostgresURL)
	redisPool := dbconn.NewRedisPool(cfg.RedisURL)
	elasticConn := dbconn.NewElastic(cfg)

	userStore := db.NewDBUserStore(dbConn)
	sessionStore := dbRedis.NewRedisSessionStore(redisPool)
	elasticProfileStore := dbElastic.NewElasticProfileStore(elasticConn, "profile")

	profileClient, profileConn, err := grpcclient.WaitForGRPC(cfg.ProfileService, pb.NewProfileServiceClient, 10, 2*time.Second)
	if err != nil {
		log.Fatalf("не удалось подключиться к ProfileService: %v", err)
	}
	defer profileConn.Close()

	authService := service.NewAuthService(userStore, sessionStore, elasticProfileStore, profileClient)
	grpcAuthHandler := authHandler.NewGrpcAuthHandler(authService)
	lis, err := net.Listen("tcp", cfg.AuthService)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	grpcServer := grpc.NewServer()
	pb.RegisterAuthServiceServer(grpcServer, grpcAuthHandler)

	log.Println("AuthService gRPC server listening on", cfg.AuthService)
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatal(err)
	}
}
