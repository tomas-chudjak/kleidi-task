package claude

import (
	"embed"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

//go:embed skills
var skillsFS embed.FS

// InstallSkills writes embedded Claude Code skill files to the project's
// .claude/skills/ directory. Overwrites existing files to propagate updates.
func InstallSkills(projectPath string) error {
	return fs.WalkDir(skillsFS, "skills", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// path is relative to embed root, e.g. "skills/new-task/SKILL.md"
		// We want to write to ".claude/skills/new-task/SKILL.md"
		targetPath := filepath.Join(projectPath, ".claude", path)

		if d.IsDir() {
			return os.MkdirAll(targetPath, 0755)
		}

		data, err := skillsFS.ReadFile(path)
		if err != nil {
			return fmt.Errorf("reading embedded skill %s: %w", path, err)
		}

		dir := filepath.Dir(targetPath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("creating skill directory %s: %w", dir, err)
		}

		if err := os.WriteFile(targetPath, data, 0644); err != nil {
			return fmt.Errorf("writing skill %s: %w", targetPath, err)
		}

		return nil
	})
}
