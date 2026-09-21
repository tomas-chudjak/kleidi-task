package core

import (
	"context"
	"fmt"
)

// OpenQuestionsSection is the heading under which unresolved decisions live.
const OpenQuestionsSection = "Open questions"

// TaskReview is everything an agent needs to critique a task description: the
// spec as it stands, the structure it is meant to have, and the instruction to
// review it against.
//
// It carries no findings — the agent produces those. Review performs no writes
// so that the author accepts a change before it lands.
type TaskReview struct {
	TaskID           int64    `json:"task_id"`
	Title            string   `json:"title"`
	Type             TaskType `json:"type"`
	Phase            string   `json:"phase,omitempty"`
	Description      string   `json:"description"`
	RequiredSections []string `json:"required_sections,omitempty"`
	MissingSections  []string `json:"missing_sections,omitempty"`
	EmptySections    []string `json:"empty_sections,omitempty"`
	OpenQuestions    string   `json:"open_questions,omitempty"`
	Instruction      string   `json:"instruction"`
}

// Review assembles the review context for a task. It never modifies the task.
func (s *TaskService) Review(ctx context.Context, id int64) (TaskReview, error) {
	task, err := s.Get(ctx, id)
	if err != nil {
		return TaskReview{}, err
	}

	review := TaskReview{
		TaskID:        task.ID,
		Title:         task.Title,
		Type:          task.Type,
		Phase:         task.Phase,
		Description:   task.Description,
		OpenQuestions: SectionContent(task.Description, OpenQuestionsSection),
	}

	if s.templates == nil {
		return TaskReview{}, fmt.Errorf("%w: templates are not configured for this project", ErrInvalidInput)
	}

	tmpl, err := s.templates.GetByType(ctx, string(task.Type))
	if err != nil {
		return TaskReview{}, fmt.Errorf("%w: no template for type '%s' — nothing to review this task against", ErrNotFound, task.Type)
	}

	review.RequiredSections = ParseSections(tmpl.Description)
	review.MissingSections = MissingSections(task.Description, review.RequiredSections)
	review.Instruction = tmpl.ReviewInstruction

	// A section that is present but empty is a gap the agent should see, and
	// is invisible to MissingSections.
	for _, section := range review.RequiredSections {
		if section == OpenQuestionsSection {
			continue // an empty Open questions is the good case
		}
		if HasSectionContent(task.Description, section) {
			continue
		}
		if contains(review.MissingSections, section) {
			continue // already reported as missing
		}
		review.EmptySections = append(review.EmptySections, section)
	}

	return review, nil
}

// contains reports whether needle is in haystack.
func contains(haystack []string, needle string) bool {
	for _, s := range haystack {
		if s == needle {
			return true
		}
	}
	return false
}
