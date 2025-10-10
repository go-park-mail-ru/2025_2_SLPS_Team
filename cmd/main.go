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

	// GET    /api/posts                    - список постов с пагинацией
	// GET    /api/posts/{id}               - получение конкретного поста
	// POST   /api/posts                    - создание поста (требует авторизации)
	// PUT    /api/posts/{id}               - обновление поста (требует авторизации)
	// DELETE /api/posts/{id}               - удаление поста (требует авторизации)
	// GET    /api/users/{userID}/posts     - посты конкретного пользователя

	// Posts routes (публичные - не требуют авторизации) [0-9] это регулярка, которая ограничивает допустимые значения. Те пропустит только цифры тк это id
	// Это валидация на уровне маршрутизации. Это фишка муршрутизатора gorilla/mux
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
