package slices

func Match[T comparable](s1, s2 []T) bool {
	set := map[T]bool{}

	for _, s := range s1 {
		set[s] = true
	}

	for _, s := range s2 {
		if _, exists := set[s]; !exists {
			return false
		}
		delete(set, s)
	}

	return len(set) == 0
}
