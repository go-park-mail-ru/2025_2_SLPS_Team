package main

import (
	"log"
	"net"
	"net/http"
	"project/cmd/dbconn"
	"project/cmd/grpcclient"
	"project/config"
	authHandler "project/internal/grpc"
	"project/internal/repository/db"
	"project/internal/repository/dbElastic"
	"project/internal/repository/dbRedis"
	"project/internal/service"
	"project/metrics"
	"project/shared/pb"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"google.golang.org/grpc"
)

func main() {
	cfg := config.NewConfig()

	authMetrics := metrics.NewGRPCMetrics("auth-service")
	dbConn := dbconn.NewPostgres(cfg.PostgresURLAuth)
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

	grpcServer := grpc.NewServer(grpc.ChainUnaryInterceptor(authHandler.ServerUnaryInterceptor(), authMetrics.UnaryServerInterceptor()))
	pb.RegisterAuthServiceServer(grpcServer, grpcAuthHandler)
	go func() {
		log.Println("AuthService gRPC server listening on", cfg.AuthService)
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
	metrics.StartHealthUpdater("auth-service", 5)
	select {}
}
