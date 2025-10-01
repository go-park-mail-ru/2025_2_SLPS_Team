package handler

import (
	"encoding/json"
	"log"
	"net/http"
	"project/repository"

	"github.com/gorilla/schema"
)

type PostsHandler struct {
	postsStore *repository.PostsStore
}

func NewPostsHandler(posts []repository.Post) *PostsHandler {
	return &PostsHandler{
		postsStore: repository.NewPostStore(posts),
	}
}

type PostsRequest struct {
	Page  int `schema:"page"`
	Limit int `schema:"limit"`
}
type PostsResponse struct {
	Posts      []repository.Post `json:"posts"`
	PagesCount int               `json:"pages"`
}

func (api *PostsHandler) PostsPaginate(w http.ResponseWriter, r *http.Request) {
	var req PostsRequest
	if err := schema.NewDecoder().Decode(&req, r.URL.Query()); err != nil {
		sendJSONError(w, "Invalid params", http.StatusBadRequest)
		return
	}

	if req.Page <= 0 || req.Limit <= 0 {
		sendJSONError(w, "Invalid params", http.StatusBadRequest)
		return
	}

	paginatedPostList, pagesCount := api.postsStore.PostsPaginatedList(req.Page, req.Limit)

	res := PostsResponse{
		Posts:      paginatedPostList,
		PagesCount: pagesCount,
	}

	if err := json.NewEncoder(w).Encode(res); err != nil {
		log.Printf("failed to write JSON response: %v", err)
	}
}
