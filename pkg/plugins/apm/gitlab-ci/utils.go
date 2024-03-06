package gitlab_ci

import "strings"

func matchAny(src []string, matchSet []string) bool {
	for _, m := range matchSet {
		for _, s := range src {
			if strings.Compare(m, s) == 0 {
				return true
			}
		}
	}

	return false
}

func splitTags(src string) (include []string, exclude []string) {
	for _, t := range strings.Split(src, ",") {
		tag := strings.TrimSpace(t)
		if tag == "" {
			continue
		}

		if strings.HasPrefix(tag, "-") {
			exclude = append(exclude, tag[1:])
			continue
		}

		include = append(include, tag)
	}

	return
}
