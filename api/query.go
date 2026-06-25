package api

import (
	"strconv"

	"github.com/gin-gonic/gin"
)

const (
	defaultPage     = 1
	defaultPageSize = 50
	maxPageSize     = 200
)

type PageQuery struct {
	Page     int `json:"page"`
	PageSize int `json:"page_size"`
	Limit    int `json:"limit"`
	Offset   int `json:"offset"`
}

func StringQuery(ctx *gin.Context, name string) *string {
	value := ctx.Query(name)
	if value == "" {
		return nil
	}
	return &value
}

func IntQuery(ctx *gin.Context, name string) (int, error) {
	value := ctx.Query(name)
	if value == "" {
		return 0, nil
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return 0, err
	}
	return parsed, nil
}

func UintQuery(ctx *gin.Context, name string) (*uint, error) {
	value := ctx.Query(name)
	if value == "" {
		return nil, nil
	}
	parsed, err := strconv.ParseUint(value, 10, 64)
	if err != nil {
		return nil, err
	}
	id := uint(parsed)
	return &id, nil
}

func PaginationQuery(ctx *gin.Context) (PageQuery, error) {
	page, err := IntQuery(ctx, "page")
	if err != nil {
		return PageQuery{}, err
	}
	pageSize, err := IntQuery(ctx, "page_size")
	if err != nil {
		return PageQuery{}, err
	}
	limit, err := IntQuery(ctx, "limit")
	if err != nil {
		return PageQuery{}, err
	}

	if page <= 0 {
		page = defaultPage
	}
	if pageSize <= 0 {
		pageSize = defaultPageSize
	}
	if limit > 0 {
		pageSize = limit
	}
	if pageSize > maxPageSize {
		pageSize = maxPageSize
	}

	return PageQuery{
		Page:     page,
		PageSize: pageSize,
		Limit:    pageSize,
		Offset:   (page - 1) * pageSize,
	}, nil
}
