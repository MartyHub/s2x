package s2x

func contains[T comparable](s []T, item T) bool {
	for _, a := range s {
		if a == item {
			return true
		}
	}

	return false
}
