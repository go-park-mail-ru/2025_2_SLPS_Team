package domain

type Post struct {
	ID              uint     `json:"id"`
	Text            string   `json:"text"`
	LikeCount       uint     `json:"likes"`
	RepostsCount    uint     `json:"reposts"`
	CommentCount    uint     `json:"comments"`
	GroupName       string   `json:"groupName"`
	CommunityAvatar string   `json:"communityAvatar"`
	PhotosPath      []string `json:"photos"`
}
type PostStore interface {
	PostsPaginatedList(page, limit int) ([]Post, int)
}
