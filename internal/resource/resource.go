// Package resource provides pure, typed transformations. It never queries data.
package resource

// Item treats a missing optional model as JSON null without calling transform.
func Item[T, R any](value *T, transform func(T) R) *R {
	if value == nil {
		return nil
	}
	result := transform(*value)
	return &result
}

// Collection always returns a non-nil slice, including for nil input.
func Collection[T, R any](items []T, transform func(T) R) []R {
	result := make([]R, len(items))
	for i, item := range items {
		result[i] = transform(item)
	}
	return result
}
