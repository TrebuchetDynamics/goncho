package limitutil

func Default(limit, defaultLimit int) int {
	if limit <= 0 {
		return defaultLimit
	}
	return limit
}

func DefaultClamped(limit, defaultLimit, maxLimit int) int {
	if limit <= 0 || limit > maxLimit {
		return defaultLimit
	}
	return limit
}
