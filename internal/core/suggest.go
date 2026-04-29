package core

import (
	"bufio"
	"context"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

// SuggestService scans source code for TODO/FIXME/HACK/XXX comments and suggests tasks.
type SuggestService struct {
	projectPath string
	taskService *TaskService
}

// NewSuggestService creates a new SuggestService for the given project.
func NewSuggestService(projectPath string, taskService *TaskService) *SuggestService {
	return &SuggestService{
		projectPath: projectPath,
		taskService: taskService,
	}
}

// commentPattern matches action tags in source code comments.
// Requires //, #, /*, or "* " prefix to avoid false positives.
var commentPattern = regexp.MustCompile(`(?:\/\/|#|\/\*|\*\s)\s*(?i)(TODO|FIXME|HACK|XXX)[:\s]+(.+)`)

// Scan analyzes the working directory and returns task suggestions.
func (s *SuggestService) Scan(ctx context.Context) ([]Suggestion, error) {
	files, err := s.listFiles(ctx)
	if err != nil {
		return nil, err
	}

	var suggestions []Suggestion
	for _, f := range files {
		found, err := s.scanFile(f)
		if err != nil {
			continue
		}
		suggestions = append(suggestions, found...)
	}

	// Deduplicate against existing tasks
	if s.taskService != nil && len(suggestions) > 0 {
		tasks, _ := s.taskService.List(ctx, ListTasksFilter{Limit: 1000})
		s.markDuplicates(suggestions, tasks)
	}

	return suggestions, nil
}

// listFiles uses git ls-files to get tracked + untracked non-ignored files.
func (s *SuggestService) listFiles(ctx context.Context) ([]string, error) {
	cmd := exec.CommandContext(ctx, "git", "ls-files", "--cached", "--others", "--exclude-standard")
	cmd.Dir = s.projectPath
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	var files []string
	scanner := bufio.NewScanner(strings.NewReader(string(out)))
	for scanner.Scan() {
		f := scanner.Text()
		if isSourceFile(f) {
			files = append(files, f)
		}
	}
	return files, nil
}

// scanFile reads a file and extracts TODO/FIXME/HACK/XXX comments.
func (s *SuggestService) scanFile(relPath string) ([]Suggestion, error) {
	absPath := filepath.Join(s.projectPath, relPath)
	cmd := exec.Command("cat", absPath)
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	var suggestions []Suggestion
	scanner := bufio.NewScanner(strings.NewReader(string(out)))
	lineNum := 0
	for scanner.Scan() {
		lineNum++
		line := scanner.Text()
		matches := commentPattern.FindStringSubmatch(line)
		if matches == nil {
			continue
		}
		kind := kindFromTag(matches[1])
		text := strings.TrimSpace(matches[2])
		if text == "" {
			continue
		}
		suggestions = append(suggestions, Suggestion{
			Kind: kind,
			Text: text,
			File: relPath,
			Line: lineNum,
		})
	}
	return suggestions, nil
}

// markDuplicates checks suggestions against existing tasks by fuzzy title matching.
func (s *SuggestService) markDuplicates(suggestions []Suggestion, tasks []Task) {
	for i := range suggestions {
		sugLower := strings.ToLower(suggestions[i].Text)
		for _, t := range tasks {
			titleLower := strings.ToLower(t.Title)
			if strings.Contains(titleLower, sugLower) || strings.Contains(sugLower, titleLower) {
				id := t.ID
				suggestions[i].ExistingTaskID = &id
				break
			}
		}
	}
}

func kindFromTag(tag string) SuggestionKind {
	switch strings.ToUpper(tag) {
	case "FIXME":
		return SuggestFixme
	case "HACK":
		return SuggestHack
	case "XXX":
		return SuggestXXX
	default:
		return SuggestTodo
	}
}

var sourceExtensions = map[string]bool{
	".go": true, ".js": true, ".ts": true, ".tsx": true, ".jsx": true,
	".py": true, ".rb": true, ".rs": true, ".java": true, ".kt": true,
	".c": true, ".cpp": true, ".h": true, ".hpp": true, ".cs": true,
	".sh": true, ".bash": true, ".zsh": true, ".fish": true,
	".sql": true, ".templ": true, ".html": true, ".css": true,
	".yaml": true, ".yml": true, ".toml": true, ".json": true,
	".md": true, ".txt": true, ".lua": true, ".php": true,
	".swift": true, ".m": true, ".scala": true, ".clj": true,
	".ex": true, ".exs": true, ".erl": true, ".zig": true,
}

func isSourceFile(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	return sourceExtensions[ext]
}

// FormatSuggestion returns a human-readable string for a suggestion.
func FormatSuggestion(s Suggestion, index int) string {
	var b strings.Builder
	b.WriteString(strconv.Itoa(index + 1))
	b.WriteString(". [")
	b.WriteString(strings.ToUpper(string(s.Kind)))
	b.WriteString("] ")
	b.WriteString(s.Text)
	b.WriteString("\n   ")
	b.WriteString(s.File)
	b.WriteString(":")
	b.WriteString(strconv.Itoa(s.Line))
	if s.ExistingTaskID != nil {
		b.WriteString("  (matches task #")
		b.WriteString(strconv.FormatInt(*s.ExistingTaskID, 10))
		b.WriteString(")")
	}
	return b.String()
}
