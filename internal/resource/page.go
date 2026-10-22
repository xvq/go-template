package resource

// Page carries query results and pagination metadata from a service.
type Page[T any] struct {
	Items    []T
	Total    int64
	Page     int
	PageSize int
}

type PageResponse[T any] struct {
	Items    []T   `json:"items"`
	Total    int64 `json:"total"`
	Page     int   `json:"page"`
	PageSize int   `json:"page_size"`
}

// Paginate preserves pagination metadata while transforming each item.
// Input metadata comes from a validated query; GORM slices do not contain it.
func Paginate[T, R any](page Page[T], transform func(T) R) PageResponse[R] {
	return PageResponse[R]{Items: Collection(page.Items, transform), Total: page.Total, Page: page.Page, PageSize: page.PageSize}
}
