package util

func Diff(from, to []string) (both, add, del []string) {
	both = []string{}
	add = []string{}
	del = []string{}
	fromDict := map[string]bool{}
	for _, path := range from {
		fromDict[path] = true
	}
	toDict := map[string]bool{}
	for _, path := range to {
		toDict[path] = true
	}
	for _, path := range from {
		if _, exists := toDict[path]; exists {
			both = append(both, path)
		} else {
			add = append(add, path)
		}
	}
	for _, path := range to {
		if _, exists := fromDict[path]; !exists {
			del = append(del, path)
		}
	}
	return
}
