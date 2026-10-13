package model

import (
	"testing"
)

func TestKnowledgeStatusValidation(t *testing.T) {
	validStatuses := []KnowledgeStatus{
		StatusObservation, StatusProposed, StatusVerified, StatusRejected, StatusSuperseded,
	}
	for _, s := range validStatuses {
		if !s.IsValid() {
			t.Errorf("expected status %s to be valid", s)
		}
	}
	if KnowledgeStatus("UNKNOWN").IsValid() {
		t.Errorf("expected UNKNOWN status to be invalid")
	}
}

func TestValidateTransition(t *testing.T) {
	// Observation -> Proposed is valid
	if err := ValidateTransition(StatusObservation, StatusProposed, false); err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	// Observation -> Verified without evidence should fail
	if err := ValidateTransition(StatusObservation, StatusVerified, false); err == nil {
		t.Errorf("expected error verifying observation without evidence")
	}

	// Observation -> Verified with evidence is valid
	if err := ValidateTransition(StatusObservation, StatusVerified, true); err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	// Proposed -> Verified without evidence should fail
	if err := ValidateTransition(StatusProposed, StatusVerified, false); err == nil {
		t.Errorf("expected error verifying proposed without evidence")
	}

	// Proposed -> Verified with evidence is valid
	if err := ValidateTransition(StatusProposed, StatusVerified, true); err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	// Proposed -> Rejected is valid
	if err := ValidateTransition(StatusProposed, StatusRejected, false); err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	// Verified -> Superseded is valid
	if err := ValidateTransition(StatusVerified, StatusSuperseded, false); err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	// Rejected -> Verified should fail
	if err := ValidateTransition(StatusRejected, StatusVerified, true); err == nil {
		t.Errorf("expected error transitioning from rejected")
	}

	// Superseded -> Verified should fail
	if err := ValidateTransition(StatusSuperseded, StatusVerified, true); err == nil {
		t.Errorf("expected error transitioning from superseded")
	}
}

func TestEdgeTypeValidation(t *testing.T) {
	if !EdgeParentOf.IsValid() {
		t.Errorf("expected PARENT_OF to be valid")
	}
	if !EdgeRelevantTo.IsValid() {
		t.Errorf("expected RELEVANT_TO to be valid")
	}
	if EdgeType("INVALID_EDGE").IsValid() {
		t.Errorf("expected INVALID_EDGE to be invalid")
	}
}
