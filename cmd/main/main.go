package main

import (
	"database/sql"
	"log"
	"net/http"
	"project/cmd/dbconn"
	"project/cmd/grpcclient"
	"project/cmd/logger"
	"project/config"
	_ "project/docs"
	"project/internal/handler"
	"project/internal/repository/db"
	"project/internal/repository/dbElastic"
	"project/internal/service"
	"project/metrics"
	"project/shared/pb"
	"time"

	"github.com/elastic/go-elasticsearch/v8"
	"github.com/gomodule/redigo/redis"
	"github.com/gorilla/mux"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	httpSwagger "github.com/swaggo/http-swagger"
	"go.uber.org/zap"
)

func NewApiRouter(logger *zap.Logger,
	dbConn *sql.DB,
	redisPool *redis.Pool,
	elasticConn *elasticsearch.Client,
	config *config.Config,
	authClient pb.AuthServiceClient,
	profileClient pb.ProfileServiceClient,
	friendClient pb.FriendServiceClient,

) *mux.Router {

	chatStore := db.NewDBChatStore(dbConn)
	messageStore := db.NewDBMessageStore(dbConn)
	postStore := db.NewDBPostStore(dbConn)
	applicationStore := db.NewDBApplicationStore(dbConn)

	elasticCommunityStore := dbElastic.NewElasticCommunityStore(elasticConn, "community_index")
	communityStore := db.NewDBCommunityStore(dbConn)
	wsHub := service.NewHub()

	chatService := service.NewChatService(authClient, profileClient, chatStore, messageStore, wsHub)
	communityService := service.NewCommunityService(communityStore, postStore, authClient, elasticCommunityStore, profileClient)
	applicationService := service.NewApplicationService(authClient, applicationStore, wsHub)
	application := handler.NewApplicationHandler(applicationService)
	auth := handler.NewAuthHandler(authClient, config)
	profile := handler.NewProfileHandler(profileClient)
	chat := handler.NewChatHandler(chatService)
	postService := service.NewPostService(postStore, authClient, communityStore, profileClient)
	middleware := handler.NewMiddlewareHandler(config)
	posts := handler.NewPostsHandler(postService)
	community := handler.NewCommunityHandler(communityService)
	wshandler := handler.NewWSHandler(wsHub)
	friend := handler.NewFriendHandler(friendClient)

	r := mux.NewRouter()
	mt := metrics.NewHTTPMetrics("main")
	r.Use(mt.HTTPMiddleware)

	r.Use(middleware.CorsMiddleware)
	r.PathPrefix("/uploads/").Handler(handler.UploadsHandler("./uploads", "/uploads/"))

	r.PathPrefix("/docs/").Handler(httpSwagger.WrapHandler)
	apiRouter := r.PathPrefix("/api").Subrouter()

	apiRouter.Use(middleware.SecureMiddleware)
	apiRouter.Use(middleware.LoggingMiddleware(logger))
	apiRouter.Use(auth.AuthMiddleware)
	apiRouter.Use(application.TempSessionMiddleware)
	apiRouter.HandleFunc("/ws", wshandler.ServeWs)
	authRouter := apiRouter.PathPrefix("/auth").Subrouter()
	authRouter.HandleFunc("/register", auth.Register).Methods("POST", "OPTIONS")
	authRouter.HandleFunc("/login", auth.Login).Methods("POST", "OPTIONS")
	authRouter.HandleFunc("/logout", auth.Logout).Methods("POST", "OPTIONS")
	authRouter.HandleFunc("/isloggedin", auth.IsLoggedInHandler).Methods("GET")

	profileRouter := apiRouter.PathPrefix("/profile").Subrouter()
	profileRouter.HandleFunc("/{id:[0-9]+}", profile.GetProfileByUserID).Methods("GET")
	profileRouter.HandleFunc("", profile.UpdateProfile).Methods("PUT", "OPTIONS")
	profileRouter.HandleFunc("/avatar", profile.UpdateAvatar).Methods("PUT", "OPTIONS")
	profileRouter.HandleFunc("/avatar", profile.DeleteAvatar).Methods("DELETE", "OPTIONS")
	profileRouter.HandleFunc("/header", profile.UpdateHeader).Methods("PUT", "OPTIONS")

	chatRouter := apiRouter.PathPrefix("/chats").Subrouter()
	chatRouter.HandleFunc("", chat.GetUserChats).Methods("GET")
	chatRouter.HandleFunc("/user/{id:[0-9]+}", chat.GetOrCreateChatWithUser).Methods("GET")
	chatRouter.HandleFunc("/{id:[0-9]+}/message", chat.CreateMessage).Methods("POST", "OPTIONS")
	chatRouter.HandleFunc("/{id:[0-9]+}/messages", chat.GetMessagesByChatId).Methods("GET")
	chatRouter.HandleFunc("/{id:[0-9]+}/last-read", chat.UpdateLastReadMessage).Methods("PUT", "OPTIONS")

	appRouter := apiRouter.PathPrefix("/applications").Subrouter()
	appRouter.HandleFunc("", application.CreateApplication).Methods("POST", "OPTIONS")
	appRouter.HandleFunc("", application.GetApplications).Methods("GET")
	appRouter.HandleFunc("/{id:[0-9]+}/text", application.UpdateApplicationText).Methods("PUT", "OPTIONS")
	appRouter.HandleFunc("/{id:[0-9]+}/status", application.UpdateApplicationStatus).Methods("PUT", "OPTIONS")

	// Posts routes (публичные - не требуют авторизации)
	apiRouter.HandleFunc("/posts", posts.PostsPaginate).Methods("GET")
	apiRouter.HandleFunc("/posts/{id:[0-9]+}", posts.GetPost).Methods("GET")
	apiRouter.HandleFunc("/users/{userID:[0-9]+}/posts", posts.GetUserPosts).Methods("GET")
	apiRouter.HandleFunc("/posts/communities/{id:[0-9]+}", posts.GetCommunityPosts).Methods("GET")

	// Posts routes (требуют авторизации)
	postsAuthRouter := apiRouter.PathPrefix("/posts").Subrouter()
	postsAuthRouter.Use(auth.AuthMiddleware)
	postsAuthRouter.HandleFunc("", posts.CreatePost).Methods("POST", "OPTIONS")
	postsAuthRouter.HandleFunc("/{id:[0-9]+}", posts.UpdatePost).Methods("PUT", "OPTIONS")
	postsAuthRouter.HandleFunc("/{id:[0-9]+}", posts.DeletePost).Methods("DELETE", "OPTIONS")
	postsAuthRouter.HandleFunc("/{id:[0-9]+}/like", posts.UpdateLikeOnPost).Methods("PUT", "OPTIONS")

	friendRouter := apiRouter.PathPrefix("/friends").Subrouter()
	friendRouter.Use(auth.AuthMiddleware)
	friendRouter.HandleFunc("", friend.GetFriends).Methods("GET")
	friendRouter.HandleFunc("/users/all", friend.GetAllUsers).Methods("GET")
	friendRouter.HandleFunc("/requests", friend.GetFriendRequests).Methods("GET")
	friendRouter.HandleFunc("/sent", friend.GetSentRequests).Methods("GET")
	friendRouter.HandleFunc("/{id:[0-9]+}", friend.SendFriendRequest).Methods("POST", "OPTIONS")
	friendRouter.HandleFunc("/{id:[0-9]+}/accept", friend.AcceptFriendRequest).Methods("PUT", "OPTIONS")
	friendRouter.HandleFunc("/{id:[0-9]+}/reject", friend.RejectFriendRequest).Methods("PUT", "OPTIONS")
	friendRouter.HandleFunc("/{id:[0-9]+}/status", friend.GetFriendshipStatus).Methods("GET")
	friendRouter.HandleFunc("/{id:[0-9]+}", friend.RemoveFriend).Methods("DELETE")
	friendRouter.HandleFunc("/{id:[0-9]+}/count", friend.CountUserRelations).Methods("GET")
	friendRouter.HandleFunc("/search", friend.SearchProfilesByFullName).Methods("GET")

	communityRouter := apiRouter.PathPrefix("/communities").Subrouter()
	communityRouter.Use(auth.AuthMiddleware)
	communityRouter.HandleFunc("/search", community.SearchCommunityByName).Methods("GET")
	communityRouter.HandleFunc("", community.CreateCommunity).Methods("POST", "OPTIONS")
	communityRouter.HandleFunc("/my", community.GetUserCommunities).Methods("GET")
	communityRouter.HandleFunc("/other", community.GetOtherCommunities).Methods("GET")
	communityRouter.HandleFunc("/users/{id:[0-9]+}", community.GetUserCommunitiesByID).Methods("GET")
	communityRouter.HandleFunc("/users/{userID:[0-9]+}/subscribed-ids", community.GetUserSubscribedCommunityIDs).Methods("GET")
	communityRouter.HandleFunc("/created", community.GetCreatedCommunities).Methods("GET")
	communityRouter.HandleFunc("/{id:[0-9]+}", community.GetCommunity).Methods("GET")
	communityRouter.HandleFunc("/{id:[0-9]+}", community.UpdateCommunity).Methods("PUT", "OPTIONS")
	communityRouter.HandleFunc("/{id:[0-9]+}", community.DeleteCommunity).Methods("DELETE", "OPTIONS")
	communityRouter.HandleFunc("/{id:[0-9]+}/subscribers", community.GetCommunitySubscribers).Methods("GET")
	communityRouter.HandleFunc("/{id:[0-9]+}/subscribe", community.Subscribe).Methods("POST", "OPTIONS")
	communityRouter.HandleFunc("/{id:[0-9]+}/unsubscribe", community.Unsubscribe).Methods("POST", "OPTIONS")
	// Отдельный endpoint для метрик Prometheus
	metricsRouter := r.PathPrefix("/metrics").Subrouter()
	metricsRouter.Handle("", promhttp.Handler()).Methods("GET")

	// Health check endpoint
	r.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status": "healthy", "service": "main-service"}`))
	}).Methods("GET")
	r.NotFoundHandler = http.HandlerFunc(handler.NotFoundHandler)

	return r
}

// @title VK API
// @version 1.0
// @description This is a VK API.
// @termsOfService http://swagger.io/terms/

// @host localhost:8080
// @BasePath /api/
func main() {

	config := config.NewConfig()
	if config.Debug {
		log.Println("Debug mode enabled")
	}
	logger := logger.NewLogger(config)
	defer logger.Sync()
	dbConn := dbconn.NewPostgres(config.PostgresURL)
	defer dbConn.Close()

	redisConn := dbconn.NewRedisPool(config.RedisURL)
	defer redisConn.Close()
	elasticConn := dbconn.NewElastic(config)

	profileClient, profileConn, err := grpcclient.WaitForGRPC(config.ProfileService, pb.NewProfileServiceClient, 10, 2*time.Second)
	if err != nil {
		log.Fatalf("не удалось подключиться к ProfileService: %v", err)
	}
	defer profileConn.Close()

	authClient, authConn, err := grpcclient.WaitForGRPC(config.AuthService, pb.NewAuthServiceClient, 10, 2*time.Second)
	if err != nil {
		log.Fatalf("не удалось подключиться к AuthService: %v", err)
	}
	defer authConn.Close()

	friendClient, friendConn, err := grpcclient.WaitForGRPC(config.FriendService, pb.NewFriendServiceClient, 10, 2*time.Second)
	if err != nil {
		log.Fatalf("не удалось подключиться к AuthService: %v", err)
	}
	defer friendConn.Close()

	apiRouter := NewApiRouter(logger, dbConn, redisConn, elasticConn, config, authClient, profileClient, friendClient)

	metrics.StartHealthUpdater("profile-service", 5)

	if err := http.ListenAndServe(":8080", apiRouter); err != nil {
		log.Fatalf("Server failed start: %v", err)
	}
}
