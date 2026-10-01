// Package paging normalizes page/page_size parameters.
package paging

const (
	// DefaultPageSize is used when the caller does not specify a page size.
	DefaultPageSize = 20
	// MaxPageSize caps the page size to protect the database.
	MaxPageSize = 100
)

// Normalize clamps page and size to valid values and returns the SQL offset.
func Normalize(page, size int32) (limit, offset int) {
	if page < 1 {
		page = 1
	}
	switch {
	case size < 1:
		size = DefaultPageSize
	case size > MaxPageSize:
		size = MaxPageSize
	}
	return int(size), int((page - 1) * size)
}
