// Package query provides helpers that compose native GORM queries.
package query

import (
	"github.com/xvq/go-template/internal/apperror"
	"gorm.io/gorm"
)

// Page carries data and pagination metadata, ready for optional item conversion.
type Page[T any] struct {
	Data       []T        `json:"data"`
	Pagination Pagination `json:"pagination"`
}

type Pagination struct {
	CurrentPage int   `json:"current_page"`
	LastPage    int64 `json:"last_page"`
	PerPage     int   `json:"per_page"`
	Total       int64 `json:"total"`
	Count       int   `json:"count"`
}

// NewPage packages query results, including results from custom count queries.
// Callers validate page and size; no HTTP-specific limits are imposed here.
func NewPage[T any](data []T, total int64, page, size int) Page[T] {
	if data == nil {
		data = make([]T, 0)
	}
	lastPage := int64(1)
	if size > 0 && total > 0 {
		lastPage = total / int64(size)
		if total%int64(size) != 0 {
			lastPage++
		}
	}
	return Page[T]{
		Data: data,
		Pagination: Pagination{
			CurrentPage: page,
			LastPage:    lastPage,
			PerPage:     size,
			Total:       total,
			Count:       len(data),
		},
	}
}

// Paginate counts matching rows and fetches one page using separate sessions.
// Set Model (or Table) and a stable Order on db before calling. Filters, scopes,
// projections and preloads remain available; this helper owns Limit and Offset.
// Counts follow GORM's Count semantics. For custom aggregate/join counts, query
// explicitly and assemble Page yourself. The two reads are not a snapshot unless
// the caller supplies a transaction with the required isolation.
// Invalid pagination returns apperror.ErrInvalidPagination before any query.
func Paginate[T any](db *gorm.DB, page, size int) (Page[T], error) {
	if page < 1 || page > 1000000 || size < 1 || size > 100 {
		return Page[T]{}, apperror.ErrInvalidPagination
	}
	var total int64
	data := make([]T, 0)
	if err := db.Session(&gorm.Session{}).Limit(-1).Offset(-1).Count(&total).Error; err != nil {
		return Page[T]{}, err
	}
	if err := db.Session(&gorm.Session{}).
		Offset((page - 1) * size).Limit(size).Find(&data).Error; err != nil {
		return Page[T]{}, err
	}
	return NewPage(data, total, page, size), nil
}
