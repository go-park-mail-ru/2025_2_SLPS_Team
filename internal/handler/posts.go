package handler

import (
    "encoding/json"
    "fmt"
    "log"
    "net/http"
    "strconv"
    "project/domain"

    "github.com/gorilla/mux"
    "github.com/gorilla/schema"
)

type PostsHandler struct {
    postStore domain.PostStore
    userStore domain.UserStore
}

func NewPostsHandler(postStore domain.PostStore, userStore domain.UserStore) *PostsHandler {
    return &PostsHandler{
        postStore: postStore,
        userStore: userStore,
    }
}

// PostsRequest - запрос для пагинации постов
type PostsRequest struct {
    Page  int `schema:"page"`
    Limit int `schema:"limit"`
}

// PostsResponse - ответ с постами и пагинацией
type PostsResponse struct {
    Posts      []domain.Post `json:"posts"`
    Page       int           `json:"page"`
    TotalPages int           `json:"totalPages"`
    HasNext    bool          `json:"hasNext"`
}

// CreatePostRequest - запрос на создание поста
type CreatePostRequest struct {
    Text            string   `json:"text" valid:"required,stringlength(1|4096)"`
    Attachments     []string `json:"attachments"`
    Photos          []string `json:"photos"`
    GroupName       string   `json:"groupName"`
    CommunityAvatar string   `json:"communityAvatar"`
}

// UpdatePostRequest - запрос на обновление поста
type UpdatePostRequest struct {
    Text            string   `json:"text" valid:"required,stringlength(1|4096)"`
    Attachments     []string `json:"attachments"`
    Photos          []string `json:"photos"`
    GroupName       string   `json:"groupName"`
    CommunityAvatar string   `json:"communityAvatar"`
}

// PostsPaginate - получение постов с пагинацией (публичный endpoint)
func (h *PostsHandler) PostsPaginate(w http.ResponseWriter, r *http.Request) {
    var req PostsRequest
    if err := schema.NewDecoder().Decode(&req, r.URL.Query()); err != nil {
        sendJSONError(w, "Invalid query parameters", http.StatusBadRequest)
        return
    }

    // Бизнес-логика: валидация параметров пагинации
    if req.Page <= 0 {
        req.Page = 1
    }
    if req.Limit <= 0 || req.Limit > 100 {
        req.Limit = 20
    }

    // Получаем посты из хранилища
    posts, totalPages, err := h.postStore.PostsPaginatedList(req.Page, req.Limit)
    if err != nil {
        log.Printf("Failed to get posts: %v", err)
        sendJSONError(w, "Internal server error", http.StatusInternalServerError)
        return
    }

    // Формируем ответ
    response := PostsResponse{
        Posts:      posts,
        Page:       req.Page,
        TotalPages: totalPages,
        HasNext:    req.Page < totalPages,
    }

    w.Header().Set("Content-Type", "application/json")
    if err := json.NewEncoder(w).Encode(response); err != nil {
        log.Printf("Failed to write JSON response: %v", err)
    }
}

// GetPost - получение конкретного поста по ID (публичный endpoint)
func (h *PostsHandler) GetPost(w http.ResponseWriter, r *http.Request) {
    vars := mux.Vars(r)
    postID, err := strconv.ParseUint(vars["id"], 10, 32)
    if err != nil {
        sendJSONError(w, "Invalid post ID", http.StatusBadRequest)
        return
    }

    // Бизнес-логика: получение поста
    post, err := h.postStore.GetPostByID(uint(postID))
    if err != nil {
        if err.Error() == "post not found" {
            sendJSONError(w, "Post not found", http.StatusNotFound)
        } else {
            log.Printf("Failed to get post: %v", err)
            sendJSONError(w, "Internal server error", http.StatusInternalServerError)
        }
        return
    }

    w.Header().Set("Content-Type", "application/json")
    if err := json.NewEncoder(w).Encode(post); err != nil {
        log.Printf("Failed to write JSON response: %v", err)
    }
}

// CreatePost - создание нового поста (требует авторизации)
func (h *PostsHandler) CreatePost(w http.ResponseWriter, r *http.Request) {
    var req CreatePostRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        sendJSONError(w, "Invalid JSON", http.StatusBadRequest)
        return
    }

    // Бизнес-логика: валидация данных
    if req.Text == "" {
        sendJSONError(w, "Post text cannot be empty", http.StatusBadRequest)
        return
    }
    if len(req.Text) > 4096 {
        sendJSONError(w, "Post text too long (max 4096 characters)", http.StatusBadRequest)
        return
    }

    // Получаем userID из контекста (установлен в auth middleware)
    userID, ok := r.Context().Value(userIDKey).(int)
    if !ok {
        sendJSONError(w, "Unauthorized", http.StatusUnauthorized)
        return
    }

    // Бизнес-логика: создание объекта поста
    post := &domain.Post{
        AuthorID:        uint(userID),
        Text:            req.Text,
        Attachments:     req.Attachments,
        PhotosPath:      req.Photos,
        GroupName:       req.GroupName,
        CommunityAvatar: req.CommunityAvatar,
        LikeCount:       0,
        RepostsCount:    0,
        CommentCount:    0,
    }

    // Сохраняем пост в хранилище
    if err := h.postStore.CreatePost(post); err != nil {
        log.Printf("Failed to create post: %v", err)
        sendJSONError(w, "Failed to create post", http.StatusInternalServerError)
        return
    }

    sendJSONSuccess(w, "Post created successfully", http.StatusCreated)
}

// UpdatePost - обновление поста (требует авторизации и проверки владельца)
func (h *PostsHandler) UpdatePost(w http.ResponseWriter, r *http.Request) {
    vars := mux.Vars(r)
    postID, err := strconv.ParseUint(vars["id"], 10, 32)
    if err != nil {
        sendJSONError(w, "Invalid post ID", http.StatusBadRequest)
        return
    }

    var req UpdatePostRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        sendJSONError(w, "Invalid JSON", http.StatusBadRequest)
        return
    }

    // Бизнес-логика: валидация данных
    if req.Text == "" {
        sendJSONError(w, "Post text cannot be empty", http.StatusBadRequest)
        return
    }
    if len(req.Text) > 4096 {
        sendJSONError(w, "Post text too long (max 4096 characters)", http.StatusBadRequest)
        return
    }

    // Получаем userID из контекста
    userID, ok := r.Context().Value(userIDKey).(int)
    if !ok {
        sendJSONError(w, "Unauthorized", http.StatusUnauthorized)
        return
    }

    // Бизнес-логика: проверяем существование поста и права доступа
    existingPost, err := h.postStore.GetPostByID(uint(postID))
    if err != nil {
        if err.Error() == "post not found" {
            sendJSONError(w, "Post not found", http.StatusNotFound)
        } else {
            log.Printf("Failed to get post: %v", err)
            sendJSONError(w, "Internal server error", http.StatusInternalServerError)
        }
        return
    }

    // Проверяем, что пользователь является автором поста
    if existingPost.AuthorID != uint(userID) {
        sendJSONError(w, "Access denied: you can only edit your own posts", http.StatusForbidden)
        return
    }

    // Обновляем данные поста
    updatedPost := &domain.Post{
        ID:              uint(postID),
        AuthorID:        uint(userID),
        Text:            req.Text,
        Attachments:     req.Attachments,
        PhotosPath:      req.Photos,
        GroupName:       req.GroupName,
        CommunityAvatar: req.CommunityAvatar,
        LikeCount:       existingPost.LikeCount,
        RepostsCount:    existingPost.RepostsCount,
        CommentCount:    existingPost.CommentCount,
        CreatedAt:       existingPost.CreatedAt,
    }

    if err := h.postStore.UpdatePost(updatedPost); err != nil {
        log.Printf("Failed to update post: %v", err)
        sendJSONError(w, "Failed to update post", http.StatusInternalServerError)
        return
    }

    sendJSONSuccess(w, "Post updated successfully", http.StatusOK)
}

// DeletePost - удаление поста (требует авторизации и проверки владельца)
func (h *PostsHandler) DeletePost(w http.ResponseWriter, r *http.Request) {
    vars := mux.Vars(r)
    postID, err := strconv.ParseUint(vars["id"], 10, 32)
    if err != nil {
        sendJSONError(w, "Invalid post ID", http.StatusBadRequest)
        return
    }

    // Получаем userID из контекста
    userID, ok := r.Context().Value(userIDKey).(int)
    if !ok {
        sendJSONError(w, "Unauthorized", http.StatusUnauthorized)
        return
    }

    // Бизнес-логика: проверяем существование поста перед удалением
    existingPost, err := h.postStore.GetPostByID(uint(postID))
    if err != nil {
        if err.Error() == "post not found" {
            sendJSONError(w, "Post not found", http.StatusNotFound)
        } else {
            log.Printf("Failed to get post: %v", err)
            sendJSONError(w, "Internal server error", http.StatusInternalServerError)
        }
        return
    }

    // Проверяем, что пользователь является автором поста
    if existingPost.AuthorID != uint(userID) {
        sendJSONError(w, "Access denied: you can only delete your own posts", http.StatusForbidden)
        return
    }

    // Удаляем пост
    if err := h.postStore.DeletePost(uint(postID), uint(userID)); err != nil {
        log.Printf("Failed to delete post: %v", err)
        sendJSONError(w, "Failed to delete post", http.StatusInternalServerError)
        return
    }

    sendJSONSuccess(w, "Post deleted successfully", http.StatusOK)
}

// GetUserPosts - получение постов конкретного пользователя (публичный endpoint)
func (h *PostsHandler) GetUserPosts(w http.ResponseWriter, r *http.Request) {
    vars := mux.Vars(r)
    userID, err := strconv.ParseUint(vars["userID"], 10, 32)
    if err != nil {
        sendJSONError(w, "Invalid user ID", http.StatusBadRequest)
        return
    }

    var req PostsRequest
    if err := schema.NewDecoder().Decode(&req, r.URL.Query()); err != nil {
        sendJSONError(w, "Invalid query parameters", http.StatusBadRequest)
        return
    }

    // Бизнес-логика: валидация параметров пагинации
    if req.Page <= 0 {
        req.Page = 1
    }
    if req.Limit <= 0 || req.Limit > 100 {
        req.Limit = 20
    }

    // Бизнес-логика: проверяем существование пользователя
    _, err = h.userStore.GetUserByID(int(userID))
    if err != nil {
        sendJSONError(w, "User not found", http.StatusNotFound)
        return
    }

    // Получаем посты пользователя
    posts, totalPages, err := h.postStore.GetPostsByUser(uint(userID), req.Page, req.Limit)
    if err != nil {
        log.Printf("Failed to get user posts: %v", err)
        sendJSONError(w, "Internal server error", http.StatusInternalServerError)
        return
    }

    // Формируем ответ
    response := PostsResponse{
        Posts:      posts,
        Page:       req.Page,
        TotalPages: totalPages,
        HasNext:    req.Page < totalPages,
    }

    w.Header().Set("Content-Type", "application/json")
    if err := json.NewEncoder(w).Encode(response); err != nil {
        log.Printf("Failed to write JSON response: %v", err)
    }
}