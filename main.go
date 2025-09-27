package main

import (
	"log"
	"project/handlers"
	"project/store"

	"net/http"

	"github.com/gorilla/mux"
)

func NewMuxRouter() *mux.Router {
	r := mux.NewRouter()
	auth := handlers.NewAuthHandler()
	posts := handlers.NewPostsHandler(store.ForkPosts)
	apiRouter := r.PathPrefix("/api").Subrouter()

	authRouter := apiRouter.PathPrefix("/auth").Subrouter()

	authRouter.Use(auth.AuthMiddleware)

	authRouter.HandleFunc("/register", auth.Register).Methods("POST")
	authRouter.HandleFunc("/login", auth.Login).Methods("POST")
	authRouter.HandleFunc("/logout", auth.Logout).Methods("POST")
	authRouter.HandleFunc("/isloggedin", auth.IsLoggedInHandler).Methods("GET")

	apiRouter.HandleFunc("/posts", posts.PostsPaginate).Methods("GET")

	staticRouter := r.PathPrefix("/static/").Subrouter()
	staticDir := http.Dir("static/")
	staticRouter.PathPrefix("/").Handler(handlers.StaticHandler(staticDir, "/static/")).Methods("GET")

	r.PathPrefix("/").HandlerFunc(handlers.SPAHandler)
	return r
}

func main() {
	r := NewMuxRouter()
	if err := http.ListenAndServe(":8080", r); err != nil {
		log.Fatalf("Server failed start: %v", err)
	}

}
