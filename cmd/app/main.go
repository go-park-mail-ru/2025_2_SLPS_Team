package main

import (
	"database/sql"
	"log"
	"net/http"
	"project/config"
	_ "project/docs"
	"project/internal/handler"
	"project/internal/repository/db"
	"project/internal/repository/dbRedis"
	"project/internal/service"

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

func NewRedis(dataSourceName string) redis.Conn {
	var err error
	redisConn, err := redis.DialURL(dataSourceName)
	if err != nil {
		log.Fatalf("cant connect to dbRedis: %v", err)
	}

	pong, err := redis.String(redisConn.Do("PING"))
	if err != nil {
		log.Fatalf("Error PING: %v", err)
	}

	log.Println("Redis connected:", pong)
	return redisConn
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

func NewApiRouter(logger *zap.Logger, dbConn *sql.DB, redisConn redis.Conn, config *config.Config) *mux.Router {

	userStore := db.NewDBUserStore(dbConn)
	sessionStore := dbRedis.NewRedisSessionStore(redisConn)
	profileStore := db.NewDBProfileStore(dbConn)
	chatStore := db.NewDBChatStore(dbConn)
	messageStore := db.NewDBMessageStore(dbConn)
	postStore := db.NewDBPostStore(dbConn)
	wsHub := service.NewHub()
	friendStore := db.NewDBFriendStore(dbConn)
	authService := service.NewAuthService(userStore, sessionStore)
	profileService := service.NewProfileService(profileStore, userStore)
	chatService := service.NewChatService(userStore, profileStore, chatStore, messageStore, wsHub)
	auth := handler.NewAuthHandler(authService, config)
	profile := handler.NewProfileHandler(profileService)
	chat := handler.NewChatHandler(chatService)
	postService := service.NewPostService(postStore, userStore)
	middleware := handler.NewMiddlewareHandler(config)
	posts := handler.NewPostsHandler(postService)
	wshandler := handler.NewWSHandler(wsHub)
	friendService := service.NewFriendService(friendStore, userStore)
	friend := handler.NewFriendHandler(friendService)

	r := mux.NewRouter()
	r.PathPrefix("/uploads/").Handler(handler.UploadsHandler("./uploads", "/uploads/"))
	r.PathPrefix("/docs/").Handler(httpSwagger.WrapHandler)
	apiRouter := r.PathPrefix("/api").Subrouter()

	apiRouter.Use(middleware.SecureMiddleware)
	apiRouter.Use(middleware.CorsMiddleware)
	apiRouter.Use(middleware.LoggingMiddleware(logger))
	apiRouter.Use(auth.AuthMiddleware)
	apiRouter.HandleFunc("/ws", wshandler.ServeWs)
	authRouter := apiRouter.PathPrefix("/auth").Subrouter()
	authRouter.HandleFunc("/register", auth.Register).Methods("POST", "OPTIONS")
	authRouter.HandleFunc("/login", auth.Login).Methods("POST", "OPTIONS")
	authRouter.HandleFunc("/logout", auth.Logout).Methods("POST", "OPTIONS")
	authRouter.HandleFunc("/isloggedin", auth.IsLoggedInHandler).Methods("GET")

	apiRouter.HandleFunc("/profile/{id:[0-9]+}", profile.GetProfileByUserID).Methods("GET")
	apiRouter.HandleFunc("/profile", profile.UpdateProfile).Methods("PUT")
	apiRouter.HandleFunc("/profile/avatar", profile.UpdateAvatar).Methods("PUT")
	apiRouter.HandleFunc("/profile/header", profile.UpdateHeader).Methods("PUT")

	chatRouter := apiRouter.PathPrefix("/chats").Subrouter()
	chatRouter.HandleFunc("", chat.GetUserChats).Methods("GET")
	chatRouter.HandleFunc("/user/{id:[0-9]+}", chat.GetOrCreateChatWithUser).Methods("GET")
	chatRouter.HandleFunc("/{id:[0-9]+}/message", chat.CreateMessage).Methods("POST")
	chatRouter.HandleFunc("/{id:[0-9]+}/messages", chat.GetMessagesByChatId).Methods("GET")

	// Posts routes (публичные - не требуют авторизации)
	apiRouter.HandleFunc("/posts", posts.PostsPaginate).Methods("GET")
	apiRouter.HandleFunc("/posts/{id:[0-9]+}", posts.GetPost).Methods("GET")
	apiRouter.HandleFunc("/users/{userID:[0-9]+}/posts", posts.GetUserPosts).Methods("GET")

	// Posts routes (требуют авторизации)
	postsAuthRouter := apiRouter.PathPrefix("/posts").Subrouter()
	postsAuthRouter.Use(auth.AuthMiddleware)
	postsAuthRouter.HandleFunc("", posts.CreatePost).Methods("POST")
	postsAuthRouter.HandleFunc("/{id:[0-9]+}", posts.UpdatePost).Methods("PUT")
	postsAuthRouter.HandleFunc("/{id:[0-9]+}", posts.DeletePost).Methods("DELETE")

	friendRouter := apiRouter.PathPrefix("/friends").Subrouter()
	friendRouter.Use(auth.AuthMiddleware)
	friendRouter.HandleFunc("", friend.GetFriends).Methods("GET")
	friendRouter.HandleFunc("/users/all", friend.GetAllUsers).Methods("GET")
	friendRouter.HandleFunc("/requests", friend.GetFriendRequests).Methods("GET")
	friendRouter.HandleFunc("/sent", friend.GetSentRequests).Methods("GET")
	friendRouter.HandleFunc("/{id:[0-9]+}", friend.SendFriendRequest).Methods("POST")
	friendRouter.HandleFunc("/{id:[0-9]+}/accept", friend.AcceptFriendRequest).Methods("PUT")
	friendRouter.HandleFunc("/{id:[0-9]+}/reject", friend.RejectFriendRequest).Methods("PUT")
	friendRouter.HandleFunc("/{id:[0-9]+}/status", friend.GetFriendshipStatus).Methods("GET")
	friendRouter.HandleFunc("/{id:[0-9]+}", friend.RemoveFriend).Methods("DELETE")
	friendRouter.HandleFunc("/{id:[0-9]+}/count", friend.CountUserRelations).Methods("GET")

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

	redisConn := NewRedis(config.RedisURL)
	defer redisConn.Close()

	apiRouter := NewApiRouter(logger, dbConn, redisConn, config)

	if err := http.ListenAndServe(":8080", apiRouter); err != nil {
		log.Fatalf("Server failed start: %v", err)
	}

}
