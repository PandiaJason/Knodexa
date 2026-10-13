package model

import "fmt"

// KnowledgeStatus represents the verification and lifecycle state of a knowledge item.
type KnowledgeStatus string

const (
	StatusObservation KnowledgeStatus = "OBSERVATION"
	StatusProposed    KnowledgeStatus = "PROPOSED"
	StatusVerified    KnowledgeStatus = "VERIFIED"
	StatusRejected    KnowledgeStatus = "REJECTED"
	StatusSuperseded  KnowledgeStatus = "SUPERSEDED"
)

// IsValid checks if the status is one of the supported knowledge statuses.
func (s KnowledgeStatus) IsValid() bool {
	switch s {
	case StatusObservation, StatusProposed, StatusVerified, StatusRejected, StatusSuperseded:
		return true
	default:
		return false
	}
}

// TaskStatus represents the execution lifecycle of a task.
type TaskStatus string

const (
	TaskStatusProposed  TaskStatus = "PROPOSED"
	TaskStatusActive    TaskStatus = "ACTIVE"
	TaskStatusBlocked   TaskStatus = "BLOCKED"
	TaskStatusCompleted TaskStatus = "COMPLETED"
	TaskStatusAbandoned TaskStatus = "ABANDONED"
)

// IsValid checks if the task status is valid.
func (ts TaskStatus) IsValid() bool {
	switch ts {
	case TaskStatusProposed, TaskStatusActive, TaskStatusBlocked, TaskStatusCompleted, TaskStatusAbandoned:
		return true
	default:
		return false
	}
}

// EvidenceType defines the classification of verification evidence.
type EvidenceType string

const (
	EvidenceTestRun       EvidenceType = "TEST_RUN"
	EvidenceBenchmark     EvidenceType = "BENCHMARK"
	EvidenceBuildTest     EvidenceType = "BUILD_TEST"
	EvidenceGitDiff       EvidenceType = "GIT_DIFF"
	EvidenceManualSignOff EvidenceType = "MANUAL_SIGN_OFF"
)

// IsValid checks if the evidence type is valid.
func (et EvidenceType) IsValid() bool {
	switch et {
	case EvidenceTestRun, EvidenceBenchmark, EvidenceBuildTest, EvidenceGitDiff, EvidenceManualSignOff:
		return true
	default:
		return false
	}
}

// ValidateTransition verifies whether a knowledge lifecycle transition is permitted.
func ValidateTransition(current, next KnowledgeStatus, hasEvidence bool) error {
	if current == next {
		return nil
	}
	switch current {
	case StatusObservation:
		if next == StatusProposed || next == StatusRejected {
			return nil
		}
		if next == StatusVerified {
			if !hasEvidence {
				return fmt.Errorf("cannot transition OBSERVATION directly to VERIFIED without supporting evidence")
			}
			return nil
		}
	case StatusProposed:
		if next == StatusVerified {
			if !hasEvidence {
				return fmt.Errorf("cannot verify PROPOSED knowledge without supporting evidence")
			}
			return nil
		}
		if next == StatusRejected {
			return nil
		}
	case StatusVerified:
		if next == StatusSuperseded {
			return nil
		}
		if next == StatusRejected {
			return nil
		}
	case StatusRejected:
		return fmt.Errorf("cannot transition from REJECTED state; create a new proposal instead")
	case StatusSuperseded:
		return fmt.Errorf("cannot transition from SUPERSEDED state; item is archived")
	}
	return fmt.Errorf("invalid transition from %s to %s", current, next)
}
