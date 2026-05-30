package eventfilter

import "path"

func Match(event string, filters []string) bool {
	if len(filters) == 0 {
		return true
	}
	for _, pattern := range filters {
		matched, err := path.Match(pattern, event)
		if err != nil {
			continue
		}
		if matched {
			return true
		}
	}
	return false
}
