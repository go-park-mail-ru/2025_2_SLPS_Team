package domain

type PaginateQueryParams struct {
	Limit int `json:"limit"`
	Page  int `json:"page"`
}

var defaultLimit = 20
var maxLimit = 100

func ValidatePaginationParams(params PaginateQueryParams) (int, int) {
	if params.Page < 1 {
		params.Page = 1
	}
	if params.Limit < 1 || params.Limit > maxLimit {
		params.Limit = defaultLimit
	}
	offset := (params.Page - 1) * params.Limit
	return offset, params.Limit
}
