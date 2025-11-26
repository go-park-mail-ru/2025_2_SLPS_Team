package main

import (
	"log"
	"net"
	"project/cmd/dbconn"
	"project/cmd/grpcclient"
	"project/config"
	friendHandler "project/internal/grpc"
	"project/internal/repository/db"
	"project/internal/repository/dbElastic"
	"project/internal/service"
	"project/shared/pb"
	"time"

	"google.golang.org/grpc"
)

func main() {
	cfg := config.NewConfig()
	//log := logger.NewLogger(cfg)

	dbConn := dbconn.NewPostgres(cfg.PostgresURL)
	elasticConn := dbconn.NewElastic(cfg)

	// Хранилища
	friendStore := db.NewDBFriendStore(dbConn)
	elasticProfileStore := dbElastic.NewElasticProfileStore(elasticConn, "profile")

	profileClient, profileConn, err := grpcclient.WaitForGRPC(cfg.ProfileService, pb.NewProfileServiceClient, 10, 2*time.Second)
	if err != nil {
		log.Fatalf("не удалось подключиться к ProfileService: %v", err)
	}
	defer profileConn.Close()

	authClient, authConn, err := grpcclient.WaitForGRPC(cfg.AuthService, pb.NewAuthServiceClient, 10, 2*time.Second)
	if err != nil {
		log.Fatalf("не удалось подключиться к AuthService: %v", err)
	}
	defer authConn.Close()

	friendService := service.NewFriendService(friendStore, authClient, elasticProfileStore, profileClient)

	lis, err := net.Listen("tcp", cfg.FriendService)
	if err != nil {
		log.Fatal("failed to listen:", err)
	}
	grpcFriendHandler := friendHandler.NewGrpcFriendHandler(friendService)
	grpcServer := grpc.NewServer()
	pb.RegisterFriendServiceServer(grpcServer, grpcFriendHandler)

	log.Println("FriendService gRPC server listening on", cfg.FriendService)
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatal(err)
	}
}
