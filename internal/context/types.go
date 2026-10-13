package context

import (
	"fmt"
	"strings"
)

// DecisionSummary represents a concise view of an architectural decision.
type DecisionSummary struct {
	ID         string  `json:"id" yaml:"id"`
	Title      string  `json:"title" yaml:"title"`
	Status     string  `json:"status" yaml:"status"`
	Confidence float64 `json:"confidence,omitempty" yaml:"confidence,omitempty"`
}

// FailureSummary represents a concise view of a previous failed approach.
type FailureSummary struct {
	ID            string `json:"id" yaml:"id"`
	Summary       string `json:"summary" yaml:"summary"`
	CausalDetails string `json:"causal_details,omitempty" yaml:"causal_details,omitempty"`
}

// LessonSummary represents a concise view of a relevant engineering lesson.
type LessonSummary struct {
	ID                 string  `json:"id" yaml:"id"`
	Title              string  `json:"title" yaml:"title"`
	SourceTask         string  `json:"source_task,omitempty" yaml:"source_task,omitempty"`
	Confidence         float64 `json:"confidence" yaml:"confidence"`
	Relevance          string  `json:"relevance" yaml:"relevance"` // high | medium | low
	VerificationStatus string  `json:"verification_status" yaml:"verification_status"`
}

// BaselineInfo represents the Git state anchor.
type BaselineInfo struct {
	Commit string `json:"commit" yaml:"commit"`
	Branch string `json:"branch,omitempty" yaml:"branch,omitempty"`
	Dirty  bool   `json:"dirty" yaml:"dirty"`
}

// TaskSummary contains basic task info for context rehydration.
type TaskSummary struct {
	ID       string `json:"id" yaml:"id"`
	ParentID string `json:"parent_id,omitempty" yaml:"parent_id,omitempty"`
	Title    string `json:"title" yaml:"title"`
	Goal     string `json:"goal" yaml:"goal"`
	Status   string `json:"status" yaml:"status"`
}

// Package represents the rehydrated engineering context for a task.
type Package struct {
	Task                 TaskSummary       `json:"task" yaml:"task"`
	AcceptedConstraints  []string          `json:"accepted_constraints" yaml:"accepted_constraints"`
	PreviousDecisions    []DecisionSummary `json:"previous_decisions,omitempty" yaml:"previous_decisions,omitempty"`
	PreviousFailures     []FailureSummary  `json:"previous_failures,omitempty" yaml:"previous_failures,omitempty"`
	RelevantLessons      []LessonSummary   `json:"relevant_lessons,omitempty" yaml:"relevant_lessons,omitempty"`
	Baseline             BaselineInfo      `json:"baseline" yaml:"baseline"`
	RequiredVerification []string          `json:"required_verification,omitempty" yaml:"required_verification,omitempty"`
	EstimatedTokens      int               `json:"estimated_tokens" yaml:"estimated_tokens"`
}

// FormatHuman returns a human-readable, token-efficient text representation.
func (p *Package) FormatHuman() string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("task: %s\n", p.Task.ID))
	sb.WriteString(fmt.Sprintf("goal:\n%s\n\n", strings.TrimSpace(p.Task.Goal)))

	if len(p.AcceptedConstraints) > 0 {
		sb.WriteString("accepted_constraints:\n")
		for _, c := range p.AcceptedConstraints {
			sb.WriteString(fmt.Sprintf("* %s\n", c))
		}
		sb.WriteString("\n")
	}

	if len(p.PreviousDecisions) > 0 {
		sb.WriteString("previous_decisions:\n")
		for _, d := range p.PreviousDecisions {
			sb.WriteString(fmt.Sprintf("* %s: %s\n", d.ID, d.Title))
		}
		sb.WriteString("\n")
	}

	if len(p.PreviousFailures) > 0 {
		sb.WriteString("previous_failures:\n")
		for _, f := range p.PreviousFailures {
			if f.CausalDetails != "" {
				sb.WriteString(fmt.Sprintf("* %s: %s (Cause: %s)\n", f.ID, f.Summary, f.CausalDetails))
			} else {
				sb.WriteString(fmt.Sprintf("* %s: %s\n", f.ID, f.Summary))
			}
		}
		sb.WriteString("\n")
	}

	if len(p.RelevantLessons) > 0 {
		sb.WriteString("relevant_lessons:\n")
		for _, l := range p.RelevantLessons {
			source := ""
			if l.SourceTask != "" {
				source = fmt.Sprintf(" from Task %s", l.SourceTask)
			}
			sb.WriteString(fmt.Sprintf("* %s%s\n    confidence: %s\n    relevance: %s\n    title: %s\n",
				l.ID, source, strings.ToLower(l.VerificationStatus), l.Relevance, l.Title))
		}
		sb.WriteString("\n")
	}

	sb.WriteString("baseline:\n")
	sb.WriteString(fmt.Sprintf("commit: %s\n", p.Baseline.Commit))
	if p.Baseline.Dirty {
		sb.WriteString("dirty: true (warning: uncommitted working tree changes)\n")
	}
	sb.WriteString("\n")

	if len(p.RequiredVerification) > 0 {
		sb.WriteString("required_verification:\n")
		for _, v := range p.RequiredVerification {
			sb.WriteString(fmt.Sprintf("* %s\n", v))
		}
	}

	return strings.TrimSpace(sb.String())
}

// EstimateTokens provides a lightweight approximation (~4 characters per token).
func EstimateTokens(text string) int {
	chars := len([]rune(text))
	if chars == 0 {
		return 0
	}
	tokens := (chars + 3) / 4
	return tokens
}
