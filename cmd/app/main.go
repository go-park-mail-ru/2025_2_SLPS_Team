package main

import (
	"database/sql"
	"log"
	"net/http"
	"project/config"
	_ "project/docs"
	"project/internal/handler"
	"project/internal/repository/db"
	"project/internal/repository/dbElasic"
	"project/internal/repository/dbRedis"
	"project/internal/service"
	"time"

	"github.com/elastic/go-elasticsearch/v8"
	"github.com/gomodule/redigo/redis"
	"github.com/gorilla/mux"
	_ "github.com/jackc/pgx/v5/stdlib"
	httpSwagger "github.com/swaggo/http-swagger"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func NewPostgres(dataSourceName string) *sql.DB {
	db, err := sql.Open("pgx", dataSourceName)
	if err != nil {
		log.Fatalf("ошибка подключения к БД: %v", err)
	}

	if err := db.Ping(); err != nil {
		log.Fatalf("ошибка ping БД: %v", err)
	}

	return db
}

func NewRedisPool(dataSourceName string) *redis.Pool {
	return &redis.Pool{
		MaxIdle:   10,
		MaxActive: 50, // 0 = без лимита
		Dial: func() (redis.Conn, error) {
			c, err := redis.DialURL(dataSourceName)
			if err != nil {
				log.Fatalf("Can't connect to Redis: %v", err)
			}
			return c, nil
		},
		TestOnBorrow: func(c redis.Conn, t time.Time) error {
			_, err := c.Do("PING")
			return err
		},
		IdleTimeout: 240 * time.Second,
	}
}
func NewElastic(config *config.Config) *elasticsearch.Client {
	cfg := elasticsearch.Config{
		Addresses: []string{
			"http://elasticsearch:" + config.ElasticPort,
		},
		Username: config.ElasticUser,
		Password: config.ElasticPassword,
	}

	es, err := elasticsearch.NewClient(cfg)
	if err != nil {
		log.Fatalf("Ошибка создания клиента: %s", err)
	}

	res, err := es.Info()
	if err != nil {
		log.Fatalf("Ошибка подключения: %s", err)
	}
	defer res.Body.Close()
	log.Println("Elasticsearch подключен:", res.Status())
	return es
}

func NewLogger(config *config.Config) *zap.Logger {
	isDebug := config.Debug
	atom := zap.NewAtomicLevel()
	incodeCfg := zap.NewProductionEncoderConfig()
	var cfg zap.Config
	if isDebug {
		atom.SetLevel(zap.DebugLevel)
		incodeCfg.EncodeTime = zapcore.ISO8601TimeEncoder
		cfg = zap.Config{
			Encoding:      "console",
			Level:         atom,
			OutputPaths:   []string{"stdout", "logs/app.log"},
			EncoderConfig: incodeCfg,
		}
	} else {
		atom.SetLevel(zap.InfoLevel)
		cfg = zap.Config{
			Encoding:      "json",
			Level:         atom,
			OutputPaths:   []string{"stdout", "logs/app.log"},
			EncoderConfig: incodeCfg,
		}
	}

	logger, err := cfg.Build()
	if err != nil {
		log.Println(err)
	}
	return logger
}

func NewApiRouter(logger *zap.Logger, dbConn *sql.DB, redisPool *redis.Pool, elasticConn *elasticsearch.Client, config *config.Config) *mux.Router {

	userStore := db.NewDBUserStore(dbConn)
	sessionStore := dbRedis.NewRedisSessionStore(redisPool)
	profileStore := db.NewDBProfileStore(dbConn)
	chatStore := db.NewDBChatStore(dbConn)
	messageStore := db.NewDBMessageStore(dbConn)
	postStore := db.NewDBPostStore(dbConn)
	applicationStore := db.NewDBApplicationStore(dbConn)
	elasticProfileStore := dbElasic.NewElasticProfileStore(elasticConn, "profile")
	communityStore := db.NewDBCommunityStore(dbConn)
	wsHub := service.NewHub()
	friendStore := db.NewDBFriendStore(dbConn)
	authService := service.NewAuthService(userStore, sessionStore, elasticProfileStore)
	profileService := service.NewProfileService(profileStore, userStore, friendStore, elasticProfileStore)
	chatService := service.NewChatService(userStore, profileStore, chatStore, messageStore, wsHub)
	communityService := service.NewCommunityService(communityStore, postStore, userStore)
	applicationService := service.NewApplicationService(userStore, applicationStore, wsHub)
	application := handler.NewApplicationHandler(applicationService)
	auth := handler.NewAuthHandler(authService, config)
	profile := handler.NewProfileHandler(profileService)
	chat := handler.NewChatHandler(chatService)
	postService := service.NewPostService(postStore, userStore)
	middleware := handler.NewMiddlewareHandler(config)
	posts := handler.NewPostsHandler(postService)
	community := handler.NewCommunityHandler(communityService)
	wshandler := handler.NewWSHandler(wsHub)
	friendService := service.NewFriendService(friendStore, userStore, elasticProfileStore)
	friend := handler.NewFriendHandler(friendService)

	r := mux.NewRouter()

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
	communityRouter.HandleFunc("", community.CreateCommunity).Methods("POST", "OPTIONS")
	communityRouter.HandleFunc("/my", community.GetUserCommunities).Methods("GET")
	communityRouter.HandleFunc("/other", community.GetOtherCommunities).Methods("GET")
	communityRouter.HandleFunc("/{id:[0-9]+}", community.GetCommunity).Methods("GET")
	communityRouter.HandleFunc("/{id:[0-9]+}", community.UpdateCommunity).Methods("PUT", "OPTIONS")
	communityRouter.HandleFunc("/{id:[0-9]+}", community.DeleteCommunity).Methods("DELETE", "OPTIONS")
	communityRouter.HandleFunc("/{id:[0-9]+}/subscribe", community.Subscribe).Methods("POST", "OPTIONS")
	communityRouter.HandleFunc("/{id:[0-9]+}/unsubscribe", community.Unsubscribe).Methods("POST", "OPTIONS")
	communityRouter.HandleFunc("/{id:[0-9]+}/subscribers/count", community.CountSubscribers).Methods("GET")
	communityRouter.HandleFunc("/{id:[0-9]+}/posts", community.GetCommunityPosts).Methods("GET")
	
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
	logger := NewLogger(config)
	defer logger.Sync()
	dbConn := NewPostgres(config.PostgresURL)
	defer dbConn.Close()

	redisConn := NewRedisPool(config.RedisURL)
	defer redisConn.Close()
	elasticConn := NewElastic(config)
	apiRouter := NewApiRouter(logger, dbConn, redisConn, elasticConn, config)

	if err := http.ListenAndServe(":8080", apiRouter); err != nil {
		log.Fatalf("Server failed start: %v", err)
	}

}
