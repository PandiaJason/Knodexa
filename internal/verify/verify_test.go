package verify

import (
	"os"
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
		ID:            "E-001",
		Title:         "Benchmarks pass",
		NodeID:        "D-001",
		EvidenceType:  model.EvidenceBenchmark,
		Command:       "fixture benchmark",
		OutputSummary: "Synthetic successful benchmark fixture",
		CreatedAt:     now,
		ExitCode:      0,
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

	if err := engine.SupersedeNode("D-001", "D-002", "SQLite replaces the previous persistence design", "agent"); err != nil {
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

func TestRejectUnsupportedEvidence(t *testing.T) {
	store := storage.NewStorage(t.TempDir())
	if _, err := store.Init("Evidence", "Regression"); err != nil {
		t.Fatal(err)
	}
	db, err := index.Open(filepath.Join(store.KnodexaDir(), "cache.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	idx := index.NewIndexer(db, store)
	engine := NewEngine(store, idx, db)
	for _, tc := range []struct {
		name string
		edit func(*model.EvidenceData)
	}{
		{"failed", func(e *model.EvidenceData) { e.ExitCode = 1 }},
		{"unrelated", func(e *model.EvidenceData) { e.NodeID = "D-other" }},
		{"no command", func(e *model.EvidenceData) { e.Command = "" }},
		{"no output", func(e *model.EvidenceData) { e.OutputSummary = "" }},
		{"no actor", func(e *model.EvidenceData) { e.VerifiedBy = "" }},
		{"bad type", func(e *model.EvidenceData) { e.EvidenceType = "INVALID" }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if err := store.SaveDecision(&model.DecisionDocument{Frontmatter: model.DecisionFrontmatter{ID: "D-test", Status: model.StatusProposed}}); err != nil {
				t.Fatal(err)
			}
			ev := &model.EvidenceData{ID: "E-test", NodeID: "D-test", EvidenceType: model.EvidenceTestRun, Command: "go test ./...", OutputSummary: "fixture success", VerifiedBy: "test", ExitCode: 0}
			tc.edit(ev)
			// Direct canonical save exercises the verifier independently of ingestion.
			if err := store.SaveEvidence(ev); err != nil {
				t.Fatal(err)
			}
			if err := engine.VerifyNode("D-test", []string{"E-test"}, "", "test"); err == nil {
				t.Fatal("unsupported evidence verified")
			}
			d, err := store.ReadDecision("D-test")
			if err != nil {
				t.Fatal(err)
			}
			if d.Frontmatter.Status != model.StatusProposed {
				t.Fatal("failed verification mutated decision")
			}
		})
	}
	for _, kind := range []model.EvidenceType{model.EvidenceTestRun, model.EvidenceBuildTest, model.EvidenceBenchmark, model.EvidenceGitDiff, model.EvidenceManualSignOff} {
		t.Run(string(kind), func(t *testing.T) {
			if err := store.SaveDecision(&model.DecisionDocument{Frontmatter: model.DecisionFrontmatter{ID: "D-test", Status: model.StatusProposed}}); err != nil {
				t.Fatal(err)
			}
			ev := &model.EvidenceData{ID: "E-test", NodeID: "D-test", EvidenceType: kind, Command: "fixture command", OutputSummary: "fixture success", VerifiedBy: "test", ExitCode: 0}
			if kind == model.EvidenceManualSignOff {
				ev.Command = ""
			}
			if err := engine.AddEvidence(ev, "test"); err != nil {
				t.Fatal(err)
			}
			if err := engine.VerifyNode("D-test", []string{"E-test"}, "", "test"); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestMissingCanonicalExitCodeCannotVerify(t *testing.T) {
	store := storage.NewStorage(t.TempDir())
	if _, err := store.Init("Evidence", ""); err != nil {
		t.Fatal(err)
	}
	db, err := index.Open(filepath.Join(store.KnodexaDir(), "cache.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	engine := NewEngine(store, index.NewIndexer(db, store), db)
	if err := store.SaveLesson(&model.LessonDocument{Frontmatter: model.LessonFrontmatter{ID: "L-test", Status: model.StatusProposed}}); err != nil {
		t.Fatal(err)
	}
	for _, suffix := range []string{"", "exit_code: null\n"} {
		raw := "id: E-test\nnode_id: L-test\nevidence_type: TEST_RUN\ncommand: test\noutput_summary: success\nverified_by: fixture\n" + suffix
		if err := os.WriteFile(filepath.Join(store.KnodexaDir(), "evidence", "E-test.yaml"), []byte(raw), 0644); err != nil {
			t.Fatal(err)
		}
		if err := engine.VerifyNode("L-test", []string{"E-test"}, "", "test"); err == nil {
			t.Fatal("omitted/null exit code accepted as successful")
		}
	}
}
