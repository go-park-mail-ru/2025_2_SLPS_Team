package main

import (
	"database/sql"
	"log"
	"net/http"
	"os"
	_ "project/docs"
	"project/internal/handler"
	"project/repository/db"

	"github.com/gorilla/mux"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/joho/godotenv"
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
func NewLogger() *zap.Logger {
	env := os.Getenv("APP_ENV")
	atom := zap.NewAtomicLevel()
	incodeCfg := zap.NewProductionEncoderConfig()
	var cfg zap.Config
	if env == "dev" {
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
	log.Println(logger)
	return logger
}

// стоит переписать через структуру, чтобы передавать все сущности
func NewApiRouter(logger *zap.Logger) *mux.Router {
	dbPath := "postgres://postgres:mysecretpassword@localhost:5432/vk?sslmode=disable"
	dbConn := NewPostgres(dbPath)
	userStore := db.NewDBUserStore(dbConn)
	sessionStore := db.NewDBSessionStore(dbConn)
	profileStore := db.NewDBProfileStore(dbConn)
	chatStore := db.NewDBChatStore(dbConn)
	messageStore := db.NewDBMessageStore(dbConn)
	postStore := db.NewDBPostStore(dbConn)

	auth := handler.NewAuthHandler(userStore, sessionStore)
	profile := handler.NewProfileHandler(profileStore, userStore)
	chat := handler.NewChatHandler(userStore, profileStore, chatStore, messageStore)
	posts := handler.NewPostsHandler(postStore, userStore)

	//hub := websocket.NewHub()
	//wsr := websocket.NewRouter()
	//message := websocket.NewWebSocketHandler(userStore)
	//wsr.Handle("send_message", )
	r := mux.NewRouter()
	r.PathPrefix("/uploads/").Handler(handler.UploadsHandler("./uploads", "/uploads/"))
	r.PathPrefix("/docs/").Handler(httpSwagger.WrapHandler)
	apiRouter := r.PathPrefix("/api").Subrouter()
	apiRouter.Use(handler.SecureMiddleware)
	apiRouter.Use(handler.CorsMiddleware)
	apiRouter.Use(auth.AuthMiddleware)
	apiRouter.Use(handler.LoggingMiddleware(logger))

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
	var err = godotenv.Load()
	if err != nil {
		log.Fatal("ошибка загрузки .env файла")
	}
	logger := NewLogger()
	defer logger.Sync()
	apiRouter := NewApiRouter(logger)

	if err := http.ListenAndServe(":8080", apiRouter); err != nil {
		log.Fatalf("Server failed start: %v", err)
	}

}
