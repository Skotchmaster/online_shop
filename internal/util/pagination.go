package util

func Calculate(page, size int) (from, limit int) {
	if page < 1 {
		page = 1
	}
	if size <= 0 || size > 100 {
		size = 10
	}
	from = (page - 1) * size
	return from, size
}
