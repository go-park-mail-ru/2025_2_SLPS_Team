package domain

type PaginateQueryParams struct {
	Limit  int `json:"limit"`
	Offset int `json:"offset"`
}
