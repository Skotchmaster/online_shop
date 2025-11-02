package util

const (
	DefaultPageSize = 10
	MaxPageSize     = 100
)

func Calculate(page, size int) (offset, limit int) {
	if page < 1 {
		page = 1
	}
	if size <= 0 || size > MaxPageSize {
		size = DefaultPageSize
	}
	offset = (page - 1) * size
	return offset, size
}
