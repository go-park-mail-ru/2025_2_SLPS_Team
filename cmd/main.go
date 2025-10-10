package main

import (
	"database/sql"
	"log"
	"net/http"
	"project/internal/handler"
	"project/repository/db"

	"github.com/gorilla/mux"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/joho/godotenv"
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

func NewApiRouter() *mux.Router {
	dbPath := "postgres://postgres:mysecretpassword@localhost:5432/vk?sslmode=disable"
	dbConn := NewPostgres(dbPath)
	userStore := db.NewDBUserStore(dbConn)
	sessionStore := db.NewDBSessionStore(dbConn)
	profileStore := db.NewDBProfileStore(dbConn)
	auth := handler.NewAuthHandler(userStore, sessionStore)
	profile := handler.NewProfileHandler(profileStore, userStore)
	r := mux.NewRouter()

	apiRouter := r.PathPrefix("/api").Subrouter()
	apiRouter.Use(handler.SecureMiddleware)
	apiRouter.Use(handler.CorsMiddleware)
	apiRouter.Use(auth.AuthMiddleware)
	authRouter := apiRouter.PathPrefix("/auth").Subrouter()

	authRouter.Use(auth.AuthMiddleware)

	authRouter.HandleFunc("/register", auth.Register).Methods("POST", "OPTIONS")
	authRouter.HandleFunc("/login", auth.Login).Methods("POST", "OPTIONS")
	authRouter.HandleFunc("/logout", auth.Logout).Methods("POST", "OPTIONS")
	authRouter.HandleFunc("/isloggedin", auth.IsLoggedInHandler).Methods("GET")
	apiRouter.HandleFunc("/profile/{id}", profile.GetProfileByUserID).Methods("GET")
	apiRouter.HandleFunc("/profile", profile.UpdateProfile).Methods("PUT")
	apiRouter.HandleFunc("/profile/avatar", profile.UpdateAvatar).Methods("PATCH")
	apiRouter.HandleFunc("/profile/header", profile.UpdateHeader).Methods("PATCH")
	authRouter.HandleFunc("/isloggedin", auth.IsLoggedInHandler).Methods("GET", "OPTIONS")

	apiRouter.HandleFunc("/posts", posts.PostsPaginate).Methods("GET")

	r.NotFoundHandler = http.HandlerFunc(handler.NotFoundHandler)

	return r
}

func main() {
	var err = godotenv.Load()
	if err != nil {
		log.Fatal("ошибка загрузки .env файла")
	}
	apiRouter := NewApiRouter()
	if err := http.ListenAndServe(":8080", apiRouter); err != nil {
		log.Fatalf("Server failed start: %v", err)
	}

}
