package core

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// taskRefPattern matches #15, klt:15, fixes #15, closes klt:15, refs #15, etc.
var taskRefPattern = regexp.MustCompile(`(?:(?:fixes|closes|refs|re)\s+)?(?:#|klt:)(\d+)`)

// GitService provides git integration without DB dependencies.
type GitService struct{}

// CommitsForTask returns commits that reference the given task ID.
func (s *GitService) CommitsForTask(ctx context.Context, projectPath string, taskID int64) ([]Commit, error) {
	if _, err := exec.LookPath("git"); err != nil {
		return nil, nil
	}

	gitDir := filepath.Join(projectPath, ".git")
	if _, err := exec.LookPath("git"); err != nil {
		return nil, nil
	}
	// Check if .git exists by trying git rev-parse
	check := exec.CommandContext(ctx, "git", "-C", projectPath, "rev-parse", "--git-dir")
	if err := check.Run(); err != nil {
		return nil, nil
	}
	_ = gitDir

	// Use git grep patterns to let git do initial filtering
	idStr := strconv.FormatInt(taskID, 10)
	grepPattern := fmt.Sprintf("#%s\\|klt:%s", idStr, idStr)

	// format: hash\x00subject\x00author\x00date(ISO)
	cmd := exec.CommandContext(ctx, "git", "-C", projectPath,
		"log", "--all",
		"--grep="+grepPattern,
		"--format=%H%x00%s%x00%an%x00%aI",
	)
	out, err := cmd.Output()
	if err != nil {
		return nil, nil // git error — silently return empty
	}

	var commits []Commit
	for _, line := range bytes.Split(bytes.TrimSpace(out), []byte("\n")) {
		if len(line) == 0 {
			continue
		}
		parts := strings.SplitN(string(line), "\x00", 4)
		if len(parts) != 4 {
			continue
		}

		// Verify this commit actually references our task ID (not just a substring match)
		matches := taskRefPattern.FindAllStringSubmatch(parts[1], -1)
		found := false
		for _, m := range matches {
			if m[1] == idStr {
				found = true
				break
			}
		}
		if !found {
			continue
		}

		date, _ := time.Parse(time.RFC3339, parts[3])
		hash := parts[0]
		shortHash := hash
		if len(hash) > 7 {
			shortHash = hash[:7]
		}

		commits = append(commits, Commit{
			Hash:      hash,
			ShortHash: shortHash,
			Message:   parts[1],
			Author:    parts[2],
			Date:      date,
		})
	}

	return commits, nil
}
