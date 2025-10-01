package main

import (
	"log"
	"net/http"
	"project/internal/handler"
	"project/repository"

	"github.com/gorilla/mux"
	"github.com/joho/godotenv"
)

func NewApiRouter() *mux.Router {
	auth := handler.NewAuthHandler(make(map[string]repository.User), make(map[string]repository.Session))
	posts := handler.NewPostsHandler(repository.ForkPosts)
	r := mux.NewRouter()

	apiRouter := r.PathPrefix("/api").Subrouter()
	apiRouter.Use(handler.SecureMiddleware)
	apiRouter.Use(handler.CorsMiddleware)

	authRouter := apiRouter.PathPrefix("/auth").Subrouter()

	authRouter.Use(auth.AuthMiddleware)

	authRouter.HandleFunc("/register", auth.Register).Methods("POST", "OPTIONS")
	authRouter.HandleFunc("/login", auth.Login).Methods("POST", "OPTIONS")
	authRouter.HandleFunc("/logout", auth.Logout).Methods("POST", "OPTIONS")
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
