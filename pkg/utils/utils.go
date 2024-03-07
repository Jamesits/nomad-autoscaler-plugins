package utils

import (
	"hash/fnv"
	"strings"
)

func MatchAny(src []string, matchSet []string) bool {
	for _, m := range matchSet {
		for _, s := range src {
			if strings.Compare(m, s) == 0 {
				return true
			}
		}
	}

	return false
}

func SplitTags(src string) (include []string, exclude []string) {
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

func Abs(n int64) int64 {
	// http://cavaliercoder.com/blog/optimized-abs-for-int64-in-go.html
	y := n >> 63
	return (n ^ y) - y
}

// FNV64a hashes using fnv32a algorithm
func FNV64a(text string) uint64 {
	algorithm := fnv.New64a()
	_, _ = algorithm.Write([]byte(text))
	return algorithm.Sum64()
}
