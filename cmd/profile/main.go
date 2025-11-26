package main

import (
	"log"
	"net"
	"project/cmd/dbconn"
	"project/config"
	profileHandler "project/internal/grpc"
	"project/internal/repository/db"
	"project/internal/repository/dbElastic"
	"project/internal/service"
	"project/shared/pb"

	"google.golang.org/grpc"
)

func main() {
	cfg := config.NewConfig()
	//log := logger.NewLogger(cfg)

	dbConn := dbconn.NewPostgres(cfg.PostgresURL)
	elasticConn := dbconn.NewElastic(cfg)

	// Хранилища
	profileStore := db.NewDBProfileStore(dbConn)
	friendStore := db.NewDBFriendStore(dbConn)
	elasticProfileStore := dbElastic.NewElasticProfileStore(elasticConn, "profile")

	profileService := service.NewProfileService(profileStore, friendStore, elasticProfileStore)

	lis, err := net.Listen("tcp", cfg.ProfileService)
	if err != nil {
		log.Fatal("failed to listen:", err)
	}
	grpcProfileHandler := profileHandler.NewGrpcProfileHandler(profileService)
	grpcServer := grpc.NewServer(grpc.UnaryInterceptor(profileHandler.ServerUnaryInterceptor()))
	pb.RegisterProfileServiceServer(grpcServer, grpcProfileHandler)

	log.Println("ProfileService gRPC server listening on", cfg.ProfileService)
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatal(err)
	}
}
