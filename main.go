package main

import (
	"log"
	"project/handlers"
	"project/store"

	"net/http"

	"github.com/gorilla/mux"
	"github.com/joho/godotenv"
)

func NewApiRouter() *mux.Router {
	auth := handlers.NewAuthHandler()
	posts := handlers.NewPostsHandler(store.ForkPosts)
	r := mux.NewRouter()

	apiRouter := r.PathPrefix("/api").Subrouter()
	apiRouter.Use(handlers.SecureMiddleware)
	apiRouter.Use(handlers.CorsMiddleware)

	authRouter := apiRouter.PathPrefix("/auth").Subrouter()

	authRouter.Use(auth.AuthMiddleware)

	authRouter.HandleFunc("/register", auth.Register).Methods("POST", "OPTIONS")
	authRouter.HandleFunc("/login", auth.Login).Methods("POST", "OPTIONS")
	authRouter.HandleFunc("/logout", auth.Logout).Methods("POST", "OPTIONS")
	authRouter.HandleFunc("/isloggedin", auth.IsLoggedInHandler).Methods("GET", "OPTIONS")

	apiRouter.HandleFunc("/posts", posts.PostsPaginate).Methods("GET")

	r.NotFoundHandler = http.HandlerFunc(handlers.NotFoundHandler)

	return r
}
func NewStaticRouter() *mux.Router {
	r := mux.NewRouter()
	r.Use(handlers.SecureMiddleware)
	staticRouter := r.PathPrefix("/static/").Subrouter()
	staticDir := http.Dir("static/")
	staticRouter.PathPrefix("/").Handler(handlers.StaticHandler(staticDir, "/static/")).Methods("GET")

	r.PathPrefix("/").HandlerFunc(handlers.SPAHandler)

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
