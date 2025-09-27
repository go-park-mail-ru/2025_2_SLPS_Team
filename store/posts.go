package store

import "sync"

type Post struct {
	ID        uint   `json:"id"`
	Text      string `json:"username"`
	LikeCount uint   `json:"like_count"`
	ImagePath string `json:"imagePath"`
}

type PostsStore struct {
	Posts []Post
	mu    sync.RWMutex
}

func (store *PostsStore) PostsPaginatedList(page, limit int) ([]Post, int) {
	store.mu.RLock()
	defer store.mu.RUnlock()

	start := (page - 1) * limit
	end := start + limit
	length := len(store.Posts)
	pagesCount := (length + limit - 1) / limit

	if start > length {
		start = length
	}

	if end > length {
		end = length
	}

	sliced := store.Posts[start:end]
	copySlice := make([]Post, len(sliced))
	copy(copySlice, sliced)

	return copySlice, pagesCount
}

var ForkPosts = []Post{
	{1, "Пост 1", 12, "/static/images/123.jpg"},
	{2, "Пост 2", 12, "/static/images/123.jpg"},
	{3, "Пост 3", 12, "/static/images/123.jpg"},
	{4, "Пост 4", 12, "/static/images/123.jpg"},
	{5, "Пост 5", 12, "/static/images/123.jpg"},
	{6, "Пост 6", 12, "/static/images/123.jpg"},
	{7, "Пост 7", 12, "/static/images/123.jpg"},
	{8, "Пост 8", 12, "/static/images/123.jpg"},
}

func NewPostStore(posts []Post) *PostsStore {
	return &PostsStore{
		Posts: posts,
		mu:    sync.RWMutex{},
	}
}
