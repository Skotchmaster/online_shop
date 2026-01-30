package util

import "strconv"

const DefaultPageSize = 20

func ParseIntDefault(s string, def int) int {
	if s == "" {
		return def
	}
	if v, err := strconv.Atoi(s); err == nil {
		return v
	}
	return def
}

func Calculate(page, size int) (offset int, limit int) {
    if page < 1 {
        page = 1
    }
    if size < 1 {
        size = DefaultPageSize
    }
    
    offset = (page - 1) * size
    limit = size
    return offset, limit
}