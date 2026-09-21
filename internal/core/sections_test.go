package core

import "testing"

const doc = "## Problem\n\nit is broken\n\n## Design\n\n### Option A\n\npick this\n\n## Open questions\n"

func TestSectionContent(t *testing.T) {
	tests := []struct {
		heading string
		want    string
	}{
		{"Problem", "it is broken"},
		{"problem", "it is broken"}, // case-insensitive
		{"Design", "### Option A\n\npick this"},
		{"Open questions", ""},
		{"Nonexistent", ""},
	}
	for _, tt := range tests {
		if got := SectionContent(doc, tt.heading); got != tt.want {
			t.Errorf("SectionContent(%q) = %q, want %q", tt.heading, got, tt.want)
		}
	}
}

func TestHasSectionContent(t *testing.T) {
	if !HasSectionContent(doc, "Problem") {
		t.Error("Problem should have content")
	}
	if HasSectionContent(doc, "Open questions") {
		t.Error("empty Open questions should report no content")
	}
	if HasSectionContent(doc, "Missing") {
		t.Error("absent section should report no content")
	}
}

func TestUpsertSectionReplacesExisting(t *testing.T) {
	got := UpsertSection(doc, "Open questions", "- Which database?")

	if SectionContent(got, "Open questions") != "- Which database?" {
		t.Errorf("section not replaced: %q", got)
	}
	// Neighbouring sections must survive untouched.
	if SectionContent(got, "Problem") != "it is broken" {
		t.Errorf("Problem was clobbered: %q", got)
	}
	if SectionContent(got, "Design") != "### Option A\n\npick this" {
		t.Errorf("Design was clobbered: %q", got)
	}
}

func TestUpsertSectionInMiddle(t *testing.T) {
	got := UpsertSection(doc, "Problem", "actually fine")

	if SectionContent(got, "Problem") != "actually fine" {
		t.Errorf("Problem not replaced: %q", got)
	}
	if SectionContent(got, "Design") != "### Option A\n\npick this" {
		t.Errorf("Design was clobbered: %q", got)
	}
	if len(ParseSections(got)) != 3 {
		t.Errorf("section count changed: %v", ParseSections(got))
	}
}

func TestUpsertSectionAppendsMissing(t *testing.T) {
	got := UpsertSection("## Problem\n\nbroken\n", "Design", "use sqlite")

	if SectionContent(got, "Design") != "use sqlite" {
		t.Errorf("Design not appended: %q", got)
	}
	if SectionContent(got, "Problem") != "broken" {
		t.Errorf("Problem lost: %q", got)
	}
}

func TestUpsertSectionOnEmptyDocument(t *testing.T) {
	got := UpsertSection("", "Design", "use sqlite")
	if got != "## Design\n\nuse sqlite\n" {
		t.Errorf("got %q", got)
	}
}

func TestUpsertSectionWithEmptyContentClearsIt(t *testing.T) {
	got := UpsertSection(doc, "Problem", "")
	if SectionContent(got, "Problem") != "" {
		t.Errorf("Problem should be empty: %q", got)
	}
	if SectionContent(got, "Design") != "### Option A\n\npick this" {
		t.Errorf("Design was clobbered: %q", got)
	}
}

func TestAppendToSection(t *testing.T) {
	got := AppendToSection(doc, "Open questions", "- Which database?")
	if SectionContent(got, "Open questions") != "- Which database?" {
		t.Errorf("first append: %q", got)
	}

	got = AppendToSection(got, "Open questions", "- Which timezone?")
	want := "- Which database?\n- Which timezone?"
	if SectionContent(got, "Open questions") != want {
		t.Errorf("second append: got %q", SectionContent(got, "Open questions"))
	}

	// Appending nothing is a no-op.
	if AppendToSection(got, "Open questions", "  ") != got {
		t.Error("empty append should not modify the document")
	}
}

func TestUpsertSectionIgnoresDeeperHeadings(t *testing.T) {
	// "### Option A" lives inside Design and must not be treated as a boundary.
	got := UpsertSection(doc, "Design", "replaced")
	if SectionContent(got, "Design") != "replaced" {
		t.Errorf("got %q", SectionContent(got, "Design"))
	}
	if SectionContent(got, "Open questions") != "" {
		t.Errorf("Open questions changed: %q", got)
	}
	if len(ParseSections(got)) != 3 {
		t.Errorf("section count changed: %v", ParseSections(got))
	}
}
