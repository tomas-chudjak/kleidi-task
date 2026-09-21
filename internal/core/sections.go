package core

import (
	"regexp"
	"strings"
)

// sectionHeadingRe matches level-2 markdown headings ("## Problem"). Deeper
// headings are content, not structure — "### Option A" inside a design section
// must not count as a required section.
var sectionHeadingRe = regexp.MustCompile(`(?m)^##[ \t]+(\S.*?)[ \t]*$`)

// ParseSections extracts the level-2 section headings from a markdown body,
// in document order, with duplicates removed.
func ParseSections(body string) []string {
	matches := sectionHeadingRe.FindAllStringSubmatch(body, -1)
	if len(matches) == 0 {
		return nil
	}
	seen := make(map[string]bool, len(matches))
	sections := make([]string, 0, len(matches))
	for _, m := range matches {
		name := strings.TrimSpace(m[1])
		key := strings.ToLower(name)
		if name == "" || seen[key] {
			continue
		}
		seen[key] = true
		sections = append(sections, name)
	}
	return sections
}

// MissingSections returns the required headings that do not appear in the
// description. Comparison is case-insensitive; order follows required.
func MissingSections(description string, required []string) []string {
	if len(required) == 0 {
		return nil
	}
	present := make(map[string]bool)
	for _, s := range ParseSections(description) {
		present[strings.ToLower(s)] = true
	}
	var missing []string
	for _, r := range required {
		if !present[strings.ToLower(r)] {
			missing = append(missing, r)
		}
	}
	return missing
}

// sectionBounds locates the named section in body. It returns the index where
// the section's content starts, the index where it ends (start of the next
// level-2 heading, or len(body)), and whether the section was found.
func sectionBounds(body, heading string) (start, end int, found bool) {
	locs := sectionHeadingRe.FindAllStringSubmatchIndex(body, -1)
	target := strings.ToLower(strings.TrimSpace(heading))

	for i, loc := range locs {
		name := strings.ToLower(strings.TrimSpace(body[loc[2]:loc[3]]))
		if name != target {
			continue
		}
		start = loc[1] // just past the heading line
		end = len(body)
		if i+1 < len(locs) {
			end = locs[i+1][0]
		}
		return start, end, true
	}
	return 0, 0, false
}

// SectionContent returns the trimmed body of the named section, or an empty
// string when the section is absent or holds nothing.
func SectionContent(body, heading string) string {
	start, end, found := sectionBounds(body, heading)
	if !found {
		return ""
	}
	return strings.TrimSpace(body[start:end])
}

// HasSectionContent reports whether the named section exists and holds
// something other than whitespace.
func HasSectionContent(body, heading string) bool {
	return SectionContent(body, heading) != ""
}

// UpsertSection replaces the body of the named section with content, appending
// the section when it does not exist yet. The rest of the document is left
// byte-for-byte intact.
func UpsertSection(body, heading, content string) string {
	content = strings.TrimSpace(content)

	start, end, found := sectionBounds(body, heading)
	if !found {
		var b strings.Builder
		b.WriteString(strings.TrimRight(body, "\n"))
		if b.Len() > 0 {
			b.WriteString("\n\n")
		}
		b.WriteString("## " + heading + "\n")
		if content != "" {
			b.WriteString("\n" + content + "\n")
		}
		return b.String()
	}

	replacement := "\n"
	if content != "" {
		replacement = "\n\n" + content + "\n"
	}
	// Keep the blank line that separated this section from the next one.
	if end < len(body) {
		replacement += "\n"
	}
	return body[:start] + replacement + body[end:]
}

// AppendToSection adds content to the end of the named section, preserving
// what is already there. Used to accumulate findings across review rounds.
func AppendToSection(body, heading, content string) string {
	content = strings.TrimSpace(content)
	if content == "" {
		return body
	}
	existing := SectionContent(body, heading)
	if existing == "" {
		return UpsertSection(body, heading, content)
	}
	return UpsertSection(body, heading, existing+"\n"+content)
}
