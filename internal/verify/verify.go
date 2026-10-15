package verify

import (
	"fmt"
	"strings"
	"time"

	"knodexa/internal/index"
	"knodexa/internal/model"
	"knodexa/internal/storage"
)

// Engine manages lifecycle state transitions and verification evidence.
type Engine struct {
	store *storage.Storage
	idx   *index.Indexer
	db    *index.DB
}

// NewEngine constructs a verification Engine.
func NewEngine(store *storage.Storage, idx *index.Indexer, db *index.DB) *Engine {
	return &Engine{
		store: store,
		idx:   idx,
		db:    db,
	}
}

// AddEvidence records new verification proof in canonical storage and links it.
func (e *Engine) AddEvidence(ev *model.EvidenceData, actor string) error {
	if ev.ID == "" {
		return fmt.Errorf("evidence ID cannot be empty")
	}
	if !ev.EvidenceType.IsValid() {
		return fmt.Errorf("invalid evidence type: %s", ev.EvidenceType)
	}
	if ev.CreatedAt.IsZero() {
		ev.CreatedAt = time.Now().UTC()
	}
	if ev.VerifiedBy == "" {
		ev.VerifiedBy = actor
	}

	if err := e.store.SaveEvidence(ev); err != nil {
		return fmt.Errorf("failed to save evidence: %w", err)
	}

	// Record link if target node provided
	if ev.NodeID != "" {
		_ = e.store.AppendLink(&model.Edge{
			FromID:    ev.NodeID,
			ToID:      ev.ID,
			Type:      model.EdgeVerifiedBy,
			CreatedAt: ev.CreatedAt,
		})
	}

	// Audit event
	_ = e.store.AppendEvent(&model.Event{
		EventID:   fmt.Sprintf("evt-%d", time.Now().UnixNano()),
		Timestamp: ev.CreatedAt,
		Actor:     actor,
		Action:    "ADD_EVIDENCE",
		TargetID:  ev.ID,
		Payload: map[string]interface{}{
			"node_id":       ev.NodeID,
			"evidence_type": string(ev.EvidenceType),
			"title":         ev.Title,
		},
	})

	return e.idx.Sync()
}

// VerifyNode promotes a proposed knowledge item (decision or lesson) to VERIFIED.
func (e *Engine) VerifyNode(nodeID string, evidenceIDs []string, notes string, actor string) error {
	if len(evidenceIDs) == 0 {
		return fmt.Errorf("verification requires at least one supporting evidence ID")
	}

	// Validate supporting evidence, not merely the existence of a file.
	for _, eid := range evidenceIDs {
		ev, err := e.store.ReadEvidence(eid)
		if err != nil {
			return fmt.Errorf("evidence %s not found: %w", eid, err)
		}
		if ev.NodeID != nodeID {
			return fmt.Errorf("evidence %s belongs to %s, not %s", eid, ev.NodeID, nodeID)
		}
		if !ev.EvidenceType.IsValid() {
			return fmt.Errorf("evidence %s has an invalid type", eid)
		}
		if strings.TrimSpace(ev.OutputSummary) == "" || strings.TrimSpace(ev.VerifiedBy) == "" {
			return fmt.Errorf("evidence %s requires an output summary and recorded actor", eid)
		}
		if ev.EvidenceType != model.EvidenceManualSignOff {
			if strings.TrimSpace(ev.Command) == "" || !ev.ExitCodeRecorded {
				return fmt.Errorf("evidence %s requires a command and explicit exit code", eid)
			}
			if ev.ExitCode != 0 {
				return fmt.Errorf("evidence %s reports failed execution (exit %d)", eid, ev.ExitCode)
			}
		}
	}

	now := time.Now().UTC()

	// Try Decision
	if strings.HasPrefix(nodeID, "D-") {
		doc, err := e.store.ReadDecision(nodeID)
		if err != nil {
			return fmt.Errorf("decision not found: %s", nodeID)
		}
		if err := model.ValidateTransition(doc.Frontmatter.Status, model.StatusVerified, true); err != nil {
			return err
		}

		doc.Frontmatter.Status = model.StatusVerified
		doc.Frontmatter.UpdatedAt = now
		doc.Frontmatter.EvidenceIDs = appendUnique(doc.Frontmatter.EvidenceIDs, evidenceIDs...)

		if err := e.store.SaveDecision(doc); err != nil {
			return fmt.Errorf("failed to update decision: %w", err)
		}

		for _, eid := range evidenceIDs {
			_ = e.store.AppendLink(&model.Edge{
				FromID:    nodeID,
				ToID:      eid,
				Type:      model.EdgeVerifiedBy,
				CreatedAt: now,
			})
		}
	} else if strings.HasPrefix(nodeID, "L-") {
		// Try Lesson
		doc, err := e.store.ReadLesson(nodeID)
		if err != nil {
			return fmt.Errorf("lesson not found: %s", nodeID)
		}
		if err := model.ValidateTransition(doc.Frontmatter.Status, model.StatusVerified, true); err != nil {
			return err
		}

		doc.Frontmatter.Status = model.StatusVerified
		doc.Frontmatter.UpdatedAt = now
		doc.Frontmatter.EvidenceIDs = appendUnique(doc.Frontmatter.EvidenceIDs, evidenceIDs...)

		if err := e.store.SaveLesson(doc); err != nil {
			return fmt.Errorf("failed to update lesson: %w", err)
		}

		for _, eid := range evidenceIDs {
			_ = e.store.AppendLink(&model.Edge{
				FromID:    nodeID,
				ToID:      eid,
				Type:      model.EdgeVerifiedBy,
				CreatedAt: now,
			})
		}
	} else {
		return fmt.Errorf("unsupported node type for verification: %s", nodeID)
	}

	// Append audit event
	_ = e.store.AppendEvent(&model.Event{
		EventID:   fmt.Sprintf("evt-%d", time.Now().UnixNano()),
		Timestamp: now,
		Actor:     actor,
		Action:    "VERIFY_KNOWLEDGE",
		TargetID:  nodeID,
		Payload: map[string]interface{}{
			"evidence_ids": evidenceIDs,
			"notes":        notes,
		},
	})

	return e.idx.Sync()
}

// RejectNode moves a knowledge item to REJECTED.
func (e *Engine) RejectNode(nodeID string, reason string, actor string) error {
	now := time.Now().UTC()

	if strings.HasPrefix(nodeID, "D-") {
		doc, err := e.store.ReadDecision(nodeID)
		if err != nil {
			return fmt.Errorf("decision not found: %s", nodeID)
		}
		if err := model.ValidateTransition(doc.Frontmatter.Status, model.StatusRejected, false); err != nil {
			return err
		}
		doc.Frontmatter.Status = model.StatusRejected
		doc.Frontmatter.UpdatedAt = now
		if err := e.store.SaveDecision(doc); err != nil {
			return err
		}
	} else if strings.HasPrefix(nodeID, "L-") {
		doc, err := e.store.ReadLesson(nodeID)
		if err != nil {
			return fmt.Errorf("lesson not found: %s", nodeID)
		}
		if err := model.ValidateTransition(doc.Frontmatter.Status, model.StatusRejected, false); err != nil {
			return err
		}
		doc.Frontmatter.Status = model.StatusRejected
		doc.Frontmatter.UpdatedAt = now
		if err := e.store.SaveLesson(doc); err != nil {
			return err
		}
	} else {
		return fmt.Errorf("unsupported node for rejection: %s", nodeID)
	}

	_ = e.store.AppendEvent(&model.Event{
		EventID:   fmt.Sprintf("evt-%d", time.Now().UnixNano()),
		Timestamp: now,
		Actor:     actor,
		Action:    "REJECT_KNOWLEDGE",
		TargetID:  nodeID,
		Payload: map[string]interface{}{
			"reason": reason,
		},
	})

	return e.idx.Sync()
}

// SupersedeNode marks an older verified knowledge item as SUPERSEDED by a newer item.
func (e *Engine) SupersedeNode(nodeID, supersededByID, reason, actor string) error {
	if nodeID == supersededByID || strings.TrimSpace(reason) == "" {
		return fmt.Errorf("supersession requires a different replacement and explicit reason")
	}
	// Read canonical replacement metadata; an index entry alone is not authority.
	switch {
	case strings.HasPrefix(nodeID, "D-") && strings.HasPrefix(supersededByID, "D-"):
		replacement, err := e.store.ReadDecision(supersededByID)
		if err != nil {
			return fmt.Errorf("replacement decision: %w", err)
		}
		if replacement.Frontmatter.Status != model.StatusVerified {
			return fmt.Errorf("replacement must be VERIFIED")
		}
	case strings.HasPrefix(nodeID, "L-") && strings.HasPrefix(supersededByID, "L-"):
		replacement, err := e.store.ReadLesson(supersededByID)
		if err != nil {
			return fmt.Errorf("replacement lesson: %w", err)
		}
		if replacement.Frontmatter.Status != model.StatusVerified {
			return fmt.Errorf("replacement must be VERIFIED")
		}
	default:
		return fmt.Errorf("supersession requires decision-to-decision or lesson-to-lesson replacement")
	}
	now := time.Now().UTC()

	if strings.HasPrefix(nodeID, "D-") {
		doc, err := e.store.ReadDecision(nodeID)
		if err != nil {
			return fmt.Errorf("decision not found: %s", nodeID)
		}
		if err := model.ValidateTransition(doc.Frontmatter.Status, model.StatusSuperseded, false); err != nil {
			return err
		}
		if doc.Frontmatter.Status != model.StatusVerified {
			return fmt.Errorf("only VERIFIED knowledge can be superseded")
		}
		doc.Frontmatter.Status = model.StatusSuperseded
		doc.Frontmatter.SupersededBy = supersededByID
		doc.Frontmatter.InvalidatedAt = &now
		doc.Frontmatter.SupersessionReason = reason
		doc.Frontmatter.UpdatedAt = now
		if err := e.store.SaveDecision(doc); err != nil {
			return err
		}
	} else if strings.HasPrefix(nodeID, "L-") {
		doc, err := e.store.ReadLesson(nodeID)
		if err != nil {
			return fmt.Errorf("lesson not found: %s", nodeID)
		}
		if err := model.ValidateTransition(doc.Frontmatter.Status, model.StatusSuperseded, false); err != nil {
			return err
		}
		if doc.Frontmatter.Status != model.StatusVerified {
			return fmt.Errorf("only VERIFIED knowledge can be superseded")
		}
		doc.Frontmatter.Status = model.StatusSuperseded
		doc.Frontmatter.SupersededBy = supersededByID
		doc.Frontmatter.InvalidatedAt = &now
		doc.Frontmatter.SupersessionReason = reason
		doc.Frontmatter.UpdatedAt = now
		if err := e.store.SaveLesson(doc); err != nil {
			return err
		}
	}

	_ = e.store.AppendLink(&model.Edge{
		FromID:    supersededByID,
		ToID:      nodeID,
		Type:      model.EdgeSupersedes,
		CreatedAt: now,
	})

	_ = e.store.AppendEvent(&model.Event{
		EventID:   fmt.Sprintf("evt-%d", time.Now().UnixNano()),
		Timestamp: now,
		Actor:     actor,
		Action:    "SUPERSEDE_KNOWLEDGE",
		TargetID:  nodeID,
		Payload: map[string]interface{}{
			"superseded_by": supersededByID,
			"reason":        reason,
		},
	})

	return e.idx.Sync()
}

func appendUnique(list []string, items ...string) []string {
	seen := make(map[string]bool)
	for _, x := range list {
		seen[x] = true
	}
	for _, x := range items {
		if !seen[x] {
			seen[x] = true
			list = append(list, x)
		}
	}
	return list
}
