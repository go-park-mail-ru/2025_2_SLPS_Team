package domain

type PaginateQueryParams struct {
	Limit int32 `json:"limit"`
	Page  int32 `json:"page"`
}

var defaultLimit = 20
var maxLimit = 100

func ValidatePaginationParams(params PaginateQueryParams) (int32, int32) {
	if params.Page < 1 {
		params.Page = 1
	}
	if params.Limit < 1 || params.Limit > int32(maxLimit) {
		params.Limit = int32(defaultLimit)
	}
	offset := (params.Page - 1) * params.Limit
	return offset, params.Limit
}
