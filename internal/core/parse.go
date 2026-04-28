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
	{"todo:", TypeTask},
	{"todo ", TypeTask},
}

// DetectTypeFromTitle checks if the title starts with a known type prefix.
// If found, returns the detected type and the title with the prefix stripped.
// If not found, returns the fallback type and the original title unchanged.
// Extra prefixes can be provided for custom task types.
func DetectTypeFromTitle(title string, fallback TaskType, extraPrefixes ...TypePrefix) (TaskType, string) {
	lower := strings.ToLower(title)

	// Check built-in prefixes first
	for _, p := range typePrefixes {
		if strings.HasPrefix(lower, p.prefix) {
			stripped := strings.TrimSpace(title[len(p.prefix):])
			if stripped != "" {
				return p.taskType, stripped
			}
		}
	}

	// Check extra (custom) prefixes
	for _, p := range extraPrefixes {
		prefix := strings.ToLower(p.Prefix) + ":"
		prefixSpace := strings.ToLower(p.Prefix) + " "
		if strings.HasPrefix(lower, prefix) {
			stripped := strings.TrimSpace(title[len(prefix):])
			if stripped != "" {
				return p.TaskType, stripped
			}
		}
		if strings.HasPrefix(lower, prefixSpace) {
			stripped := strings.TrimSpace(title[len(prefixSpace):])
			if stripped != "" {
				return p.TaskType, stripped
			}
		}
	}

	return fallback, title
}

// TypePrefix maps a title prefix to a task type.
type TypePrefix struct {
	Prefix   string
	TaskType TaskType
}
