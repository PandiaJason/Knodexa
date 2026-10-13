package verify

import (
	"path/filepath"
	"testing"
	"time"

	"knodexa/internal/index"
	"knodexa/internal/model"
	"knodexa/internal/storage"
)

func TestVerificationLifecycle(t *testing.T) {
	tmpDir := t.TempDir()
	store := storage.NewStorage(tmpDir)
	_, _ = store.Init("Knodexa Verification", "Test Engine")

	dbPath := filepath.Join(tmpDir, ".knodexa", "cache.db")
	db, err := index.Open(dbPath)
	if err != nil {
		t.Fatalf("failed to open db: %v", err)
	}
	defer db.Close()

	indexer := index.NewIndexer(db, store)
	_ = indexer.Sync()

	engine := NewEngine(store, indexer, db)

	// Create proposed decision
	now := time.Now().UTC()
	dec := &model.DecisionDocument{
		Frontmatter: model.DecisionFrontmatter{
			ID:        "D-001",
			Title:     "Use In-Memory Cache",
			Status:    model.StatusProposed,
			TaskID:    "T.1",
			CreatedAt: now,
			UpdatedAt: now,
		},
		Body: "Rationale...",
	}
	if err := store.SaveDecision(dec); err != nil {
		t.Fatalf("failed to save decision: %v", err)
	}
	_ = indexer.Sync()

	// 1. Verify without evidence must fail
	if err := engine.VerifyNode("D-001", []string{}, "Notes", "agent"); err == nil {
		t.Errorf("expected error verifying decision without evidence")
	}

	// 2. Add evidence
	ev := &model.EvidenceData{
		ID:           "E-001",
		Title:        "Benchmarks pass",
		NodeID:       "D-001",
		EvidenceType: model.EvidenceBenchmark,
		CreatedAt:    now,
		ExitCode:     0,
	}
	if err := engine.AddEvidence(ev, "agent"); err != nil {
		t.Fatalf("failed to add evidence: %v", err)
	}

	// 3. Verify with valid evidence succeeds
	if err := engine.VerifyNode("D-001", []string{"E-001"}, "Verified via benchmark", "agent"); err != nil {
		t.Fatalf("verify node failed: %v", err)
	}

	// Check updated node in DB
	n, err := db.GetNode("D-001")
	if err != nil {
		t.Fatalf("failed to get node: %v", err)
	}
	if n.Status != string(model.StatusVerified) {
		t.Errorf("expected status VERIFIED, got %s", n.Status)
	}

	// 4. Supersede with D-002
	dec2 := &model.DecisionDocument{
		Frontmatter: model.DecisionFrontmatter{
			ID:        "D-002",
			Title:     "Use SQLite instead of memory cache",
			Status:    model.StatusVerified,
			TaskID:    "T.1",
			CreatedAt: now,
			UpdatedAt: now,
		},
		Body: "Better persistence.",
	}
	_ = store.SaveDecision(dec2)
	_ = indexer.Sync()

	if err := engine.SupersedeNode("D-001", "D-002", "agent"); err != nil {
		t.Fatalf("supersede failed: %v", err)
	}

	nAfterSupersede, _ := db.GetNode("D-001")
	if nAfterSupersede.Status != string(model.StatusSuperseded) {
		t.Errorf("expected status SUPERSEDED, got %s", nAfterSupersede.Status)
	}

	// 5. Audit events
	events, err := store.ReadEvents()
	if err != nil {
		t.Fatalf("failed to read events: %v", err)
	}
	if len(events) < 3 {
		t.Errorf("expected at least 3 audit events, got %d", len(events))
	}
}
