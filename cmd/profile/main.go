package main

import (
	"log"
	"net"
	"net/http"
	"project/cmd/dbconn"
	"project/config"
	profileHandler "project/internal/grpc"
	"project/internal/repository/db"
	"project/internal/repository/dbElastic"
	"project/internal/service"
	"project/metrics"
	"project/shared/pb"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"google.golang.org/grpc"
)

func main() {
	cfg := config.NewConfig()
	profileMetrics := metrics.NewGRPCMetrics("profile-service")

	dbConn := dbconn.NewPostgres(cfg.PostgresURLProfile)
	elasticConn := dbconn.NewElastic(cfg)

	profileStore := db.NewDBProfileStore(dbConn)
	friendStore := db.NewDBFriendStore(dbConn)
	elasticProfileStore := dbElastic.NewElasticProfileStore(elasticConn, "profile")

	profileService := service.NewProfileService(profileStore, friendStore, elasticProfileStore)

	lis, err := net.Listen("tcp", cfg.ProfileService)
	if err != nil {
		log.Fatal("failed to listen:", err)
	}
	grpcProfileHandler := profileHandler.NewGrpcProfileHandler(profileService)
	grpcServer := grpc.NewServer(grpc.ChainUnaryInterceptor(profileHandler.ServerUnaryInterceptor(), profileMetrics.UnaryServerInterceptor()))
	pb.RegisterProfileServiceServer(grpcServer, grpcProfileHandler)
	go func() {
		log.Println("ProfileService gRPC server listening on", cfg.ProfileService)
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

	metrics.StartHealthUpdater("profile-service", 5)
	select {}
}
