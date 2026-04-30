package core

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"
)

// HookEvent represents a task lifecycle event.
type HookEvent string

const (
	EventTaskCreate   HookEvent = "task.create"
	EventTaskUpdate   HookEvent = "task.update"
	EventTaskComplete HookEvent = "task.complete"
	EventTaskDelete   HookEvent = "task.delete"
	EventTaskArchive  HookEvent = "task.archive"
)

// Hook defines a script to run on a task event.
type Hook struct {
	ID          int    `json:"id"`
	Event       HookEvent `json:"event"`
	Command     string    `json:"command"`
	Description string    `json:"description,omitempty"`
}

// HooksConfig is the on-disk format for .tasks/hooks.json.
type HooksConfig struct {
	Hooks []Hook `json:"hooks"`
}

// HookExecution records the result of a hook execution.
type HookExecution struct {
	Hook       Hook      `json:"hook"`
	Output     string    `json:"output"`
	Success    bool      `json:"success"`
	DurationMs int64     `json:"duration_ms"`
	ExecutedAt time.Time `json:"executed_at"`
}

// HookService manages project-level hooks.
type HookService struct {
	projectPath string
	mu          sync.RWMutex
}

// NewHookService creates a HookService for a project.
func NewHookService(projectPath string) *HookService {
	return &HookService{projectPath: projectPath}
}

func (s *HookService) configPath() string {
	return filepath.Join(s.projectPath, ".tasks", "hooks.json")
}

// List returns all configured hooks.
func (s *HookService) List() ([]Hook, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	cfg, err := s.load()
	if err != nil {
		return nil, err
	}
	return cfg.Hooks, nil
}

// Add adds a new hook and saves to disk.
func (s *HookService) Add(hook Hook) (Hook, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	cfg, _ := s.load()

	// Assign next ID
	maxID := 0
	for _, h := range cfg.Hooks {
		if h.ID > maxID {
			maxID = h.ID
		}
	}
	hook.ID = maxID + 1

	cfg.Hooks = append(cfg.Hooks, hook)
	if err := s.save(cfg); err != nil {
		return Hook{}, err
	}
	return hook, nil
}

// Remove removes a hook by ID.
func (s *HookService) Remove(id int) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	cfg, _ := s.load()
	found := false
	filtered := make([]Hook, 0, len(cfg.Hooks))
	for _, h := range cfg.Hooks {
		if h.ID == id {
			found = true
			continue
		}
		filtered = append(filtered, h)
	}
	if !found {
		return fmt.Errorf("hook %d: %w", id, ErrNotFound)
	}
	cfg.Hooks = filtered
	return s.save(cfg)
}

// Fire executes all hooks matching the event asynchronously.
// Returns immediately — hook execution happens in background goroutines.
func (s *HookService) Fire(event HookEvent, task Task) {
	s.mu.RLock()
	cfg, err := s.load()
	s.mu.RUnlock()
	if err != nil {
		return
	}

	for _, hook := range cfg.Hooks {
		if hook.Event != event {
			continue
		}
		go s.execute(hook, event, task)
	}
}

// execute runs a single hook with task context.
func (s *HookService) execute(hook Hook, event HookEvent, task Task) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	taskJSON, _ := json.Marshal(task)

	cmd := exec.CommandContext(ctx, "sh", "-c", hook.Command)
	cmd.Dir = s.projectPath
	cmd.Stdin = bytes.NewReader(taskJSON)
	cmd.Env = append(os.Environ(),
		fmt.Sprintf("KVT_EVENT=%s", event),
		fmt.Sprintf("KVT_TASK_ID=%d", task.ID),
		fmt.Sprintf("KVT_TASK_TITLE=%s", task.Title),
		fmt.Sprintf("KVT_TASK_TYPE=%s", task.Type),
		fmt.Sprintf("KVT_TASK_STATUS=%s", task.Status),
		fmt.Sprintf("KVT_TASK_PRIORITY=%d", task.Priority),
	)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	start := time.Now()
	err := cmd.Run()
	duration := time.Since(start).Milliseconds()

	output := stdout.String()
	if stderr.Len() > 0 {
		if output != "" {
			output += "\n"
		}
		output += stderr.String()
	}

	if err != nil {
		slog.Warn("hook execution failed",
			"hook_id", hook.ID,
			"event", event,
			"task_id", task.ID,
			"command", hook.Command,
			"duration_ms", duration,
			"error", err,
			"output", output,
		)
	} else {
		slog.Debug("hook executed",
			"hook_id", hook.ID,
			"event", event,
			"task_id", task.ID,
			"duration_ms", duration,
		)
	}
}

func (s *HookService) load() (HooksConfig, error) {
	data, err := os.ReadFile(s.configPath())
	if err != nil {
		if os.IsNotExist(err) {
			return HooksConfig{}, nil
		}
		return HooksConfig{}, err
	}

	var cfg HooksConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return HooksConfig{}, fmt.Errorf("parsing hooks.json: %w", err)
	}
	return cfg, nil
}

func (s *HookService) save(cfg HooksConfig) error {
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.configPath(), data, 0644)
}
