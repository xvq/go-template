// Package resource provides pure, typed transformations. It never queries data.
package resource

import "github.com/xvq/go-template/internal/util/query"

// Resource reuses one stateless item transformation for every output shape.
// Construct with New and treat the resulting resource as read-only.
type Resource[T, R any] struct {
	transform func(T) R
}

func New[T, R any](transform func(T) R) Resource[T, R] {
	return Resource[T, R]{transform: transform}
}

// Item treats a missing optional model as JSON null without calling transform.
func (r Resource[T, R]) Item(value *T) *R {
	if value == nil {
		return nil
	}
	result := r.transform(*value)
	return &result
}

// Collection always returns a non-nil slice, including for nil input.
func (r Resource[T, R]) Collection(items []T) []R {
	result := make([]R, len(items))
	for i, item := range items {
		result[i] = r.transform(item)
	}
	return result
}

// Page transforms already loaded items and preserves query pagination metadata.
// Empty collections are serialized as [] and out-of-range pages retain Total.
func (r Resource[T, R]) Page(page query.Page[T]) query.Page[R] {
	return query.Page[R]{
		Data:       r.Collection(page.Data),
		Pagination: page.Pagination,
	}
}
