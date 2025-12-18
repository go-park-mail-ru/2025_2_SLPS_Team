package main

import (
	"log"
	"net"
	"net/http"
	"project/cmd/dbconn"
	"project/cmd/grpcclient"
	"project/config"
	friendHandler "project/internal/grpc"
	"project/internal/repository/db"
	"project/internal/repository/dbElastic"
	"project/internal/service"
	"project/metrics"
	"project/shared/pb"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"google.golang.org/grpc"
)

func main() {
	cfg := config.NewConfig()
	friendMetrics := metrics.NewGRPCMetrics("friend-service")

	dbConn := dbconn.NewPostgres(cfg.PostgresURLFriend)
	elasticConn := dbconn.NewElastic(cfg)

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
	grpcServer := grpc.NewServer(grpc.ChainUnaryInterceptor(friendHandler.ServerUnaryInterceptor(), friendMetrics.UnaryServerInterceptor()))
	pb.RegisterFriendServiceServer(grpcServer, grpcFriendHandler)
	go func() {
		log.Println("FriendService gRPC server listening on", cfg.FriendService)
		if err := grpcServer.Serve(lis); err != nil {
			log.Fatal(err)
		}

	}()

	mux := http.NewServeMux()

	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status": "healthy"}`))
	})

	mux.Handle("/metrics", promhttp.Handler())

	go func() {
		if err := http.ListenAndServe(":8080", mux); err != nil {
			panic(err)
		}
	}()
	metrics.StartHealthUpdater("friend-service", 5)
	select {}
}
