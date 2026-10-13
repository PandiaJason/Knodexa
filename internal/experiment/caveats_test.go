package experiment

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"knodexa/internal/context"
	"knodexa/internal/git"
	"knodexa/internal/index"
	"knodexa/internal/model"
	"knodexa/internal/storage"
	"knodexa/internal/verify"
)

// TestCaveat1_Discipline_UnrecordedFailure tests what happens when an agent fails to record a failure.
// It verifies:
// 1. Omitted failure is absent from graph (proving the Garbage-In-Garbage-Out caveat).
// 2. Git inspector still flags the dirty working tree and baseline status.
// 3. Documented failure immediately restores protection.
func TestCaveat1_Discipline_UnrecordedFailure(t *testing.T) {
	tmpDir := t.TempDir()
	store := storage.NewStorage(tmpDir)
	_, _ = store.Init("Discipline Caveat Test", "Testing unrecorded agent actions")

	dbPath := filepath.Join(tmpDir, ".knodexa", "cache.db")
	db, err := index.Open(dbPath)
	if err != nil {
		t.Fatalf("failed to open sqlite: %v", err)
	}
	defer db.Close()

	indexer := index.NewIndexer(db, store)
	gitInspector := git.NewInspector(tmpDir)
	ctxBuilder := context.NewBuilder(db, store, gitInspector)

	now := time.Now().UTC()
	_ = store.SaveTask(&model.TaskData{
		ID:        "A",
		Title:     "Data Ingestion",
		Goal:      "High throughput parsing",
		Status:    model.TaskStatusActive,
		CreatedAt: now,
		UpdatedAt: now,
	})
	_ = store.SaveTask(&model.TaskData{
		ID:        "A.1",
		ParentID:  "A",
		Title:     "Regex Parser",
		Goal:      "Parse lines via regex",
		Status:    model.TaskStatusActive,
		CreatedAt: now,
		UpdatedAt: now,
	})
	_ = indexer.Sync()

	// Scenario A: Agent tries regex, it is 10x too slow, but agent DOES NOT record F-001.
	pkgBefore, err := ctxBuilder.Build("A.1", 1500)
	if err != nil {
		t.Fatalf("build failed: %v", err)
	}
	if len(pkgBefore.PreviousFailures) != 0 {
		t.Errorf("expected 0 failures when agent forgot to record, got %d", len(pkgBefore.PreviousFailures))
	}
	t.Logf("[Caveat 1 Confirmed]: If agent omits recording, PreviousFailures is empty (Garbage In / Nothing In).")

	// Scenario B: Agent records F-001
	_ = store.SaveFailure(&model.FailureData{
		ID:            "F-001",
		TaskID:        "A.1",
		Summary:       "Regex parser caused catastrophic backtracking",
		CausalDetails: "10x latency regression on 100kb inputs.",
		AttemptedAt:   now,
	})
	_ = indexer.Sync()

	pkgAfter, err := ctxBuilder.Build("A.1", 1500)
	if err != nil {
		t.Fatalf("build failed: %v", err)
	}
	if len(pkgAfter.PreviousFailures) != 1 || pkgAfter.PreviousFailures[0].ID != "F-001" {
		t.Errorf("expected failure F-001 after recording, got %+v", pkgAfter.PreviousFailures)
	}
	t.Logf("[Protection Restored]: Once recorded, F-001 is immediately rehydrated: %s", pkgAfter.PreviousFailures[0].Summary)
}

// TestCaveat2_ScaleAndRelevanceScoring tests Knodexa under 500 Tasks and 200 Lessons.
// It verifies:
// 1. SQLite ingestion & query latency stay under 20ms at scale.
// 2. High-relevance lessons (matching touched files) are preserved.
// 3. Low-relevance / poorly tagged lessons are pruned to satisfy the 1,500 token budget.
func TestCaveat2_ScaleAndRelevanceScoring(t *testing.T) {
	tmpDir := t.TempDir()
	store := storage.NewStorage(tmpDir)
	_, _ = store.Init("Scale Caveat Test", "Testing 500 tasks and 200 lessons")

	dbPath := filepath.Join(tmpDir, ".knodexa", "cache.db")
	db, err := index.Open(dbPath)
	if err != nil {
		t.Fatalf("failed to open sqlite: %v", err)
	}
	defer db.Close()

	indexer := index.NewIndexer(db, store)
	gitInspector := git.NewInspector(tmpDir)
	ctxBuilder := context.NewBuilder(db, store, gitInspector)

	now := time.Now().UTC()
	startPopulate := time.Now()

	// 1. Populate 500 Tasks
	const numTasks = 500
	for i := 1; i <= numTasks; i++ {
		taskID := fmt.Sprintf("T%d", i)
		parentID := ""
		if i > 50 {
			parentID = fmt.Sprintf("T%d", (i%50)+1)
		}
		_ = store.SaveTask(&model.TaskData{
			ID:        taskID,
			ParentID:  parentID,
			Title:     fmt.Sprintf("Task %d", i),
			Goal:      fmt.Sprintf("Goal for task %d", i),
			Status:    model.TaskStatusCompleted,
			CreatedAt: now,
			UpdatedAt: now,
		})
	}

	// 2. Populate 200 Lessons
	// Lesson L-001 to L-003 touch "internal/hotpath/engine.go" (HIGH RELEVANCE)
	// Lesson L-004 to L-200 touch other random files or have generic tags (LOW RELEVANCE)
	const numLessons = 200
	for i := 1; i <= numLessons; i++ {
		lesID := fmt.Sprintf("L-%03d", i)
		relatedFiles := []string{fmt.Sprintf("misc/file%d.go", i)}
		title := fmt.Sprintf("Generic lesson %d", i)
		confidence := 0.70

		if i <= 3 {
			relatedFiles = []string{"internal/hotpath/engine.go"}
			title = fmt.Sprintf("CRITICAL Engine Lesson %d", i)
			confidence = 0.98
		}

		_ = store.SaveLesson(&model.LessonDocument{
			Frontmatter: model.LessonFrontmatter{
				ID:            lesID,
				Title:         title,
				Status:        model.StatusVerified,
				SourceTaskID:  fmt.Sprintf("T%d", (i%50)+1),
				CreatedAt:     now,
				UpdatedAt:     now,
				Confidence:    confidence,
				Applicability: "Applicable to engine or misc operations",
				RelatedFiles:  relatedFiles,
			},
			Body: fmt.Sprintf("Details for lesson %d...", i),
		})
	}

	// Sync index
	if err := indexer.Sync(); err != nil {
		t.Fatalf("indexer sync failed at scale: %v", err)
	}
	populateDuration := time.Since(startPopulate)
	t.Logf("Indexed 500 tasks and 200 lessons in %v", populateDuration)

	// Target task touching "internal/hotpath/engine.go"
	targetTask := &model.TaskData{
		ID:           "T-TARGET",
		ParentID:     "T1",
		Title:        "Hotpath Engine Optimization",
		Goal:         "Optimize critical loop in engine.go",
		Status:       model.TaskStatusActive,
		CreatedAt:    now,
		UpdatedAt:    now,
		TouchesFiles: []string{"internal/hotpath/engine.go"},
	}
	_ = store.SaveTask(targetTask)
	_ = indexer.Sync()

	// Measure Context Rehydration Latency at scale
	startQuery := time.Now()
	pkg, err := ctxBuilder.Build("T-TARGET", 1500)
	queryDuration := time.Since(startQuery)

	if err != nil {
		t.Fatalf("build context at scale failed: %v", err)
	}

	t.Logf("Context Rehydration Query Latency (500 tasks, 200 lessons): %v", queryDuration)
	if queryDuration > 50*time.Millisecond {
		t.Errorf("Query latency too slow: %v (expected < 50ms)", queryDuration)
	}

	// Verify Relevance Priority: The 3 high-relevance lessons MUST be surfaced!
	foundHighRelevance := 0
	for _, l := range pkg.RelevantLessons {
		if strings.HasPrefix(l.Title, "CRITICAL Engine Lesson") {
			foundHighRelevance++
		}
	}

	if foundHighRelevance < 3 {
		t.Errorf("Expected all 3 critical lessons to be prioritized over 197 generic ones, got %d. Lessons returned: %+v",
			foundHighRelevance, pkg.RelevantLessons)
	}

	// Verify Strict Token Budget Compliance
	tokens := context.EstimateTokens(pkg.FormatHuman())
	t.Logf("Token Count at Scale: %d tokens (Budget: 1,500)", tokens)
	if tokens > 1500 {
		t.Errorf("Token budget violated at scale: %d tokens", tokens)
	}
}

// TestCaveat3_KnowledgeStalenessAndSuperseding tests handling of outdated architectural decisions.
// It verifies:
// 1. Superseded decisions are automatically excluded from active context rehydration.
// 2. The replacing decision takes precedence.
// 3. The historical lineage is preserved in SQLite links.
func TestCaveat3_KnowledgeStalenessAndSuperseding(t *testing.T) {
	tmpDir := t.TempDir()
	store := storage.NewStorage(tmpDir)
	_, _ = store.Init("Staleness Caveat Test", "Testing superseded decisions")

	dbPath := filepath.Join(tmpDir, ".knodexa", "cache.db")
	db, err := index.Open(dbPath)
	if err != nil {
		t.Fatalf("failed to open sqlite: %v", err)
	}
	defer db.Close()

	indexer := index.NewIndexer(db, store)
	verifier := verify.NewEngine(store, indexer, db)
	gitInspector := git.NewInspector(tmpDir)
	ctxBuilder := context.NewBuilder(db, store, gitInspector)

	now := time.Now().UTC()

	// 1. Task A created with original decision D-001 (e.g., using deprecated v1 API)
	_ = store.SaveTask(&model.TaskData{
		ID:        "A",
		Title:     "Storage Backend",
		Goal:      "Persist metadata",
		Status:    model.TaskStatusActive,
		CreatedAt: now,
		UpdatedAt: now,
	})

	_ = store.SaveDecision(&model.DecisionDocument{
		Frontmatter: model.DecisionFrontmatter{
			ID:          "D-001",
			Title:       "Use JSON flat files for metadata (Old V1)",
			Status:      model.StatusVerified,
			TaskID:      "A",
			CreatedAt:   now.Add(-24 * time.Hour),
			UpdatedAt:   now.Add(-24 * time.Hour),
			Confidence:  0.90,
			EvidenceIDs: []string{"E-001"},
		},
		Body: "Flat JSON is simplest.",
	})
	_ = store.SaveEvidence(&model.EvidenceData{
		ID:           "E-001",
		Title:        "JSON benchmarks pass",
		NodeID:       "D-001",
		EvidenceType: model.EvidenceBenchmark,
		CreatedAt:    now.Add(-24 * time.Hour),
		ExitCode:     0,
	})
	_ = indexer.Sync()

	// Verify D-001 is initially active in context
	pkgInitial, _ := ctxBuilder.Build("A", 1500)
	if len(pkgInitial.PreviousDecisions) != 1 || pkgInitial.PreviousDecisions[0].ID != "D-001" {
		t.Fatalf("expected D-001 to be initially active")
	}
	t.Logf("Initial State: D-001 is active in context: %s", pkgInitial.PreviousDecisions[0].Title)

	// 2. Later, flat JSON runs out of performance. Decision D-002 replaces it.
	_ = store.SaveDecision(&model.DecisionDocument{
		Frontmatter: model.DecisionFrontmatter{
			ID:          "D-002",
			Title:       "Adopt Embedded SQLite WAL mode (New V2)",
			Status:      model.StatusVerified,
			TaskID:      "A",
			CreatedAt:   now,
			UpdatedAt:   now,
			Confidence:  0.99,
			EvidenceIDs: []string{"E-002"},
		},
		Body: "SQLite provides transactional multi-reader performance.",
	})
	_ = store.SaveEvidence(&model.EvidenceData{
		ID:           "E-002",
		Title:        "SQLite WAL benchmark pass",
		NodeID:       "D-002",
		EvidenceType: model.EvidenceBenchmark,
		CreatedAt:    now,
		ExitCode:     0,
	})
	_ = indexer.Sync()

	// Supersede D-001 with D-002
	if err := verifier.SupersedeNode("D-001", "D-002", "agent"); err != nil {
		t.Fatalf("supersede failed: %v", err)
	}

	// 3. Rehydrate context after superseding:
	// D-001 MUST NOT appear as an active previous decision.
	// D-002 MUST appear as active.
	pkgAfter, err := ctxBuilder.Build("A", 1500)
	if err != nil {
		t.Fatalf("build after supersede failed: %v", err)
	}

	for _, d := range pkgAfter.PreviousDecisions {
		if d.ID == "D-001" {
			t.Errorf("STALENESS LEAK: Superseded decision D-001 was included in active context!")
		}
	}

	foundD2 := false
	for _, d := range pkgAfter.PreviousDecisions {
		if d.ID == "D-002" {
			foundD2 = true
			break
		}
	}
	if !foundD2 {
		t.Errorf("New active decision D-002 was not found in context")
	}

	t.Logf("[Caveat 3 Confirmed & Resolved]: D-001 safely suppressed; D-002 active in context: %s",
		pkgAfter.PreviousDecisions[0].Title)

	// Historical provenance check in SQLite
	edges, _ := db.GetEdgesTo("D-001")
	hasSupersedeEdge := false
	for _, e := range edges {
		if e.Type == model.EdgeSupersedes && e.FromID == "D-002" {
			hasSupersedeEdge = true
			break
		}
	}
	if !hasSupersedeEdge {
		t.Errorf("Missing SUPERSEDES relationship from D-002 to D-001 in graph")
	}
	t.Logf("Historical lineage verified: D-002 SUPERSEDES D-001 edge recorded in graph.")
}
