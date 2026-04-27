package core

import "strings"

// typePrefixes maps title prefixes to task types.
// Checked case-insensitively. Both "BUG: title" and "bug title" work.
var typePrefixes = []struct {
	prefix   string
	taskType TaskType
}{
	{"bug:", TypeBug},
	{"bug ", TypeBug},
	{"feature:", TypeFeature},
	{"feature ", TypeFeature},
	{"feat:", TypeFeature},
	{"feat ", TypeFeature},
	{"hotfix:", TypeHotfix},
	{"hotfix ", TypeHotfix},
	{"task:", TypeTask},
	{"task ", TypeTask},
}

// DetectTypeFromTitle checks if the title starts with a known type prefix.
// If found, returns the detected type and the title with the prefix stripped.
// If not found, returns the fallback type and the original title unchanged.
func DetectTypeFromTitle(title string, fallback TaskType) (TaskType, string) {
	lower := strings.ToLower(title)
	for _, p := range typePrefixes {
		if strings.HasPrefix(lower, p.prefix) {
			stripped := strings.TrimSpace(title[len(p.prefix):])
			if stripped != "" {
				return p.taskType, stripped
			}
		}
	}
	return fallback, title
}
