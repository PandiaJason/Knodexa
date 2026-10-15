package context

import (
	"fmt"
	"knodexa/internal/approach"
	"strings"
)

// DecisionSummary represents a concise view of an architectural decision.
type DecisionSummary struct {
	SourceTask  string   `json:"source_task,omitempty" yaml:"source_task,omitempty"`
	EvidenceIDs []string `json:"evidence_ids,omitempty" yaml:"evidence_ids,omitempty"`
	Summary     string   `json:"summary,omitempty" yaml:"summary,omitempty"`
	Rationale   string   `json:"rationale,omitempty" yaml:"rationale,omitempty"`
	ID          string   `json:"id" yaml:"id"`
	Title       string   `json:"title" yaml:"title"`
	Status      string   `json:"status" yaml:"status"`
	Confidence  float64  `json:"confidence,omitempty" yaml:"confidence,omitempty"`
}

// FailureSummary represents a concise view of a previous failed approach.
type FailureSummary struct {
	SourceTask    string `json:"source_task,omitempty" yaml:"source_task,omitempty"`
	ID            string `json:"id" yaml:"id"`
	Summary       string `json:"summary" yaml:"summary"`
	CausalDetails string `json:"causal_details,omitempty" yaml:"causal_details,omitempty"`
}

// LessonSummary represents a concise view of a relevant engineering lesson.
type LessonSummary struct {
	MatchReason        string   `json:"match_reason,omitempty" yaml:"match_reason,omitempty"`
	EvidenceIDs        []string `json:"evidence_ids,omitempty" yaml:"evidence_ids,omitempty"`
	Description        string   `json:"description" yaml:"description"`
	Applicability      string   `json:"applicability" yaml:"applicability"`
	ID                 string   `json:"id" yaml:"id"`
	Title              string   `json:"title" yaml:"title"`
	SourceTask         string   `json:"source_task,omitempty" yaml:"source_task,omitempty"`
	Confidence         float64  `json:"confidence" yaml:"confidence"`
	Relevance          string   `json:"relevance" yaml:"relevance"` // high | medium | low
	VerificationStatus string   `json:"verification_status" yaml:"verification_status"`
}

// DeferredMemory is a bounded discovery pointer, not dropped knowledge content.
type DeferredMemory struct {
	ID         string `json:"id" yaml:"id"`
	Kind       string `json:"kind" yaml:"kind"`
	SourceTask string `json:"source_task,omitempty" yaml:"source_task,omitempty"`
	Reason     string `json:"reason" yaml:"reason"`
}

// BaselineInfo represents the Git state anchor.
type BaselineInfo struct {
	StoredCommit string `json:"stored_commit,omitempty" yaml:"stored_commit,omitempty"`
	Mismatch     bool   `json:"mismatch,omitempty" yaml:"mismatch,omitempty"`
	Commit       string `json:"commit" yaml:"commit"`
	Branch       string `json:"branch,omitempty" yaml:"branch,omitempty"`
	Dirty        bool   `json:"dirty" yaml:"dirty"`
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
	DeferredMemory       []DeferredMemory    `json:"deferred_memory,omitempty" yaml:"deferred_memory,omitempty"`
	DeferredCount        int                 `json:"deferred_count,omitempty" yaml:"deferred_count,omitempty"`
	ApproachSelection    *approach.Selection `json:"approach_selection" yaml:"approach_selection"`
	ContextBoundary      *Boundary           `json:"context_boundary,omitempty" yaml:"context_boundary,omitempty"`
	ApproachStatus       *approach.Report    `json:"approach_status,omitempty" yaml:"approach_status,omitempty"`
	EstablishedApproach  []approach.Contract `json:"established_approach,omitempty" yaml:"established_approach,omitempty"`
	DetailLevel          string              `json:"detail_level" yaml:"detail_level"`
	DetailHint           string              `json:"detail_hint,omitempty" yaml:"detail_hint,omitempty"`
	ContextPruned        bool                `json:"context_pruned" yaml:"context_pruned"`
	TokenBudget          int                 `json:"token_budget" yaml:"token_budget"`
	BudgetExceeded       bool                `json:"budget_exceeded" yaml:"budget_exceeded"`
	Task                 TaskSummary         `json:"task" yaml:"task"`
	AcceptedConstraints  []string            `json:"accepted_constraints" yaml:"accepted_constraints"`
	PreviousDecisions    []DecisionSummary   `json:"previous_decisions,omitempty" yaml:"previous_decisions,omitempty"`
	PreviousFailures     []FailureSummary    `json:"previous_failures,omitempty" yaml:"previous_failures,omitempty"`
	RelevantLessons      []LessonSummary     `json:"relevant_lessons,omitempty" yaml:"relevant_lessons,omitempty"`
	Baseline             BaselineInfo        `json:"baseline" yaml:"baseline"`
	RequiredVerification []string            `json:"required_verification,omitempty" yaml:"required_verification,omitempty"`
	EstimatedTokens      int                 `json:"estimated_tokens" yaml:"estimated_tokens"`
}

// FormatHuman returns a human-readable, token-efficient text representation.
func (p *Package) FormatHuman() string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("task: %s\n", p.Task.ID))
	sb.WriteString("memory_policy: agreements, constraints and acceptance pinned; failures retained; lessons advisory\n")
	if p.DeferredCount > 0 {
		ids := []string{}
		for i, item := range p.DeferredMemory {
			if i >= 3 {
				break
			}
			ids = append(ids, item.ID)
		}
		fmt.Fprintf(&sb, "deferred_memory: %d omitted for budget; sample=%v; expand with knodexa show <id>\n", p.DeferredCount, ids)
	}
	if p.ApproachSelection != nil {
		sb.WriteString(fmt.Sprintf("approach_selection: %s sources=%v candidates=%v excluded=%v review_required=%t\n", p.ApproachSelection.Mode, p.ApproachSelection.Sources, p.ApproachSelection.Candidates, p.ApproachSelection.ExcludedSources, p.ApproachSelection.ReviewRequired))
	}
	if p.ContextBoundary != nil {
		sb.WriteString(fmt.Sprintf("context_boundary: from=%s to=%s discard_sources=%v\n%s\n", p.ContextBoundary.PreviousTask, p.ContextBoundary.TargetTask, p.ContextBoundary.DiscardSources, p.ContextBoundary.Instruction))
	}
	sb.WriteString(fmt.Sprintf("goal:\n%s\n\n", strings.TrimSpace(p.Task.Goal)))

	if len(p.EstablishedApproach) > 0 {
		sb.WriteString("established_approach:\n")
		for _, contract := range p.EstablishedApproach {
			for _, binding := range contract.Approach.ReferenceReconciliations {
				fmt.Fprintf(&sb, "  reviewed reference: %s version=%d %s (%s) -> %s (%s): %s\n", binding.SourceTask, binding.SourceVersion, binding.OriginalPath, binding.OriginalSHA256, binding.CurrentPath, binding.CurrentSHA256, binding.Reason)
			}
			sb.WriteString(fmt.Sprintf("* from %s version=%d: %s\n", contract.SourceTask, contract.Approach.Version, contract.Approach.Summary))
			for _, rule := range contract.Approach.Rules {
				sb.WriteString("  rule: " + rule + "\n")
			}
			for _, ref := range contract.Approach.References {
				sb.WriteString(fmt.Sprintf("  reference: %s (sha256 %s)\n", ref.Path, ref.SHA256))
			}
			for _, skill := range contract.Approach.SkillFiles {
				sb.WriteString("  skill: " + skill + "\n")
			}
			for _, check := range contract.Approach.Checks {
				if check.ASTKind != "" {
					sb.WriteString(fmt.Sprintf("  AST check: %s %s %s\n", check.Path, check.ASTKind, check.Symbol))
					continue
				}
				sb.WriteString(fmt.Sprintf("  check: %s require=%q forbid=%q\n", check.Path, check.MustContain, check.MustNotContain))
			}
		}
		sb.WriteString("Preserve these approaches; record rationale and evidence for deliberate deviations.\n\n")
	}
	if p.ApproachStatus != nil {
		sb.WriteString("approach_check:\n")
		for _, result := range p.ApproachStatus.Results {
			if !result.Passed {
				sb.WriteString(fmt.Sprintf("* FAIL [%s] %s: %s\n", result.SourceTask, result.Path, result.Detail))
			}
		}
		sb.WriteString(fmt.Sprintf("structural_checks_passed: %t\nsemantic_review_required: %t\n\n", p.ApproachStatus.StructuralChecksPassed, p.ApproachStatus.SemanticReviewRequired))
	}
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
			if d.Rationale == "" && d.Summary != "" && d.Summary != d.Title {
				sb.WriteString(fmt.Sprintf("  rule: %s\n", d.Summary))
			}
			if d.Rationale != "" {
				sb.WriteString(fmt.Sprintf("  rationale: %s\n", d.Rationale))
			}
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
			sb.WriteString(fmt.Sprintf("    applicability: %s\n    guidance: %s\n", l.Applicability, l.Description))
		}
		sb.WriteString("\n")
	}

	sb.WriteString("baseline:\n")
	sb.WriteString(fmt.Sprintf("commit: %s\n", p.Baseline.Commit))
	if p.Baseline.StoredCommit != "" {
		sb.WriteString(fmt.Sprintf("stored_commit: %s\nmismatch: %t\n", p.Baseline.StoredCommit, p.Baseline.Mismatch))
	}
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

	if p.DetailHint != "" {
		sb.WriteString("\n" + p.DetailHint + "\n")
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

// Boundary scopes remembered conventions when a conversation changes focus.
type Boundary struct {
	PreviousTask   string   `json:"previous_task,omitempty"`
	TargetTask     string   `json:"target_task"`
	DiscardSources []string `json:"discard_sources,omitempty"`
	Instruction    string   `json:"instruction"`
}
