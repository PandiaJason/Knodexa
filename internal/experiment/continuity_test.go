package experiment

import (
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

// TestContinuityExperiment reproduces the multi-task context switch experiment described in Section 14:
// Task A (A.1 -> A.2) -> Context switch to Task B (B.1 -> B.2 -> B.3) -> Return to Task A (A.3)
func TestContinuityExperiment(t *testing.T) {
	tmpDir := t.TempDir()
	store := storage.NewStorage(tmpDir)
	_, err := store.Init("Continuity Benchmark", "Testing agent continuity across task boundaries")
	if err != nil {
		t.Fatalf("failed to init storage: %v", err)
	}

	dbPath := filepath.Join(tmpDir, ".knodexa", "cache.db")
	db, err := index.Open(dbPath)
	if err != nil {
		t.Fatalf("failed to open sqlite: %v", err)
	}
	defer db.Close()

	indexer := index.NewIndexer(db, store)
	_ = indexer.Sync()

	gitInspector := git.NewInspector(tmpDir)
	verifier := verify.NewEngine(store, indexer, db)
	ctxBuilder := context.NewBuilder(db, store, gitInspector)

	now := time.Now().UTC()

	// ============================================================
	// PHASE 1: Task A (A.1, A.2)
	// ============================================================
	taskA := &model.TaskData{
		ID:          "A",
		Title:       "High-Throughput Priority Queue",
		Goal:        "Implement concurrent priority queue for event processing",
		Status:      model.TaskStatusActive,
		CreatedAt:   now,
		UpdatedAt:   now,
		Constraints: []string{"Preserve O(log N) heap invariant", "Zero background goroutine leakage"},
	}
	_ = store.SaveTask(taskA)

	taskA1 := &model.TaskData{
		ID:           "A.1",
		ParentID:     "A",
		Title:        "Baseline Heap Implementation",
		Goal:         "Evaluate heap representations",
		Status:       model.TaskStatusCompleted,
		CreatedAt:    now,
		UpdatedAt:    now,
		TouchesFiles: []string{"internal/queue/heap.go"},
	}
	_ = store.SaveTask(taskA1)

	// Decision D-001: 4-ary heap
	decD1 := &model.DecisionDocument{
		Frontmatter: model.DecisionFrontmatter{
			ID:          "D-001",
			Title:       "Use 4-ary cache-conscious min-heap",
			Status:      model.StatusVerified,
			TaskID:      "A.1",
			CreatedAt:   now,
			UpdatedAt:   now,
			Confidence:  0.95,
			EvidenceIDs: []string{"E-001"},
		},
		Body: "4-ary heap reduces tree depth and leverages cache lines effectively.",
	}
	_ = store.SaveDecision(decD1)

	// Failure F-001: Linked list failed
	failF1 := &model.FailureData{
		ID:            "F-001",
		TaskID:        "A.1",
		Summary:       "Linked-list heap failed allocation benchmark",
		CausalDetails: "Pointer chasing caused 40% throughput drop and heavy GC pressure.",
		AttemptedAt:   now,
	}
	_ = store.SaveFailure(failF1)

	// Subtask A.2
	taskA2 := &model.TaskData{
		ID:           "A.2",
		ParentID:     "A",
		Title:        "Concurrent Synchronization",
		Goal:         "Add lock-free or striped concurrency",
		Status:       model.TaskStatusCompleted,
		CreatedAt:    now,
		UpdatedAt:    now,
		TouchesFiles: []string{"internal/queue/heap.go"},
	}
	_ = store.SaveTask(taskA2)

	_ = indexer.Sync()

	// ============================================================
	// PHASE 2: Context Switch to Task B (B.1 -> B.2 -> B.3)
	// ============================================================
	taskB := &model.TaskData{
		ID:        "B",
		Title:     "Network Frame Parser",
		Goal:      "Parse binary telemetry protocol frames",
		Status:    model.TaskStatusActive,
		CreatedAt: now,
		UpdatedAt: now,
	}
	_ = store.SaveTask(taskB)

	taskB2 := &model.TaskData{
		ID:           "B.2",
		ParentID:     "B",
		Title:        "Zero-Copy Buffer Pool Optimization",
		Goal:         "Optimize frame memory allocation",
		Status:       model.TaskStatusCompleted,
		CreatedAt:    now,
		UpdatedAt:    now,
		TouchesFiles: []string{"internal/buffer/pool.go"},
	}
	_ = store.SaveTask(taskB2)

	// Task B.2 produces Lesson L-017
	lesL17 := &model.LessonDocument{
		Frontmatter: model.LessonFrontmatter{
			ID:            "L-017",
			Title:         "Retain backing buffer pointer during zero-copy slicing",
			Status:        model.StatusProposed,
			SourceTaskID:  "B.2",
			CreatedAt:     now,
			UpdatedAt:     now,
			Confidence:    0.92,
			Applicability: "When pooling byte slices or buffer instances",
			Tags:          []string{"memory", "buffer", "concurrency"},
			RelatedFiles:  []string{"internal/buffer/pool.go", "internal/queue/heap.go"},
		},
		Body: "Returning buffer to pool while sub-slices are referenced causes race conditions and memory corruption.",
	}
	_ = store.SaveLesson(lesL17)

	// Add evidence E-010 and verify L-017
	evE10 := &model.EvidenceData{
		ID:            "E-010",
		Title:         "Buffer Race Test Suite Pass",
		NodeID:        "L-017",
		EvidenceType:  model.EvidenceTestRun,
		CreatedAt:     now,
		Command:       "go test -race ./internal/buffer/...",
		ExitCode:      0,
		OutputSummary: "Synthetic race-test success fixture",
		VerifiedBy:    "agent",
	}
	if err := verifier.AddEvidence(evE10, "agent"); err != nil {
		t.Fatalf("failed to add evidence: %v", err)
	}
	if err := verifier.VerifyNode("L-017", []string{"E-010"}, "Verified race-free buffer recycling", "agent"); err != nil {
		t.Fatalf("failed to verify lesson L-017: %v", err)
	}

	_ = indexer.Sync()

	// ============================================================
	// PHASE 3: Return to Task A (Task A.3)
	// ============================================================
	taskA3 := &model.TaskData{
		ID:           "A.3",
		ParentID:     "A",
		Title:        "Buffer Pooling and Bulk Ingestion",
		Goal:         "Add buffer pooling to reduce allocations during bulk event ingestion",
		Status:       model.TaskStatusActive,
		CreatedAt:    now,
		UpdatedAt:    now,
		TouchesFiles: []string{"internal/queue/heap.go"},
		RequiredVerification: []string{
			"go test -v -race ./...",
			"go test -bench=BenchmarkBulkIngest ./...",
		},
	}
	_ = store.SaveTask(taskA3)
	_ = indexer.Sync()

	// ============================================================
	// EXPERIMENT MEASUREMENTS
	// ============================================================

	// 1. Rehydrate Context for Task A.3
	pkg, err := ctxBuilder.Build("A.3", 1500)
	if err != nil {
		t.Fatalf("rehydrate context failed: %v", err)
	}

	// Constraint Preservation: Must inherit parent constraints
	hasHeapInvariant := false
	hasGoroutineConstraint := false
	for _, c := range pkg.AcceptedConstraints {
		if strings.Contains(c, "Preserve O(log N) heap invariant") {
			hasHeapInvariant = true
		}
		if strings.Contains(c, "Zero background goroutine leakage") {
			hasGoroutineConstraint = true
		}
	}
	if !hasHeapInvariant || !hasGoroutineConstraint {
		t.Errorf("Constraint Preservation Failed: constraints missing from rehydration: %+v", pkg.AcceptedConstraints)
	}

	// Architectural Consistency: Must remember Decision D-001
	hasDecisionD1 := false
	for _, d := range pkg.PreviousDecisions {
		if d.ID == "D-001" {
			hasDecisionD1 = true
			break
		}
	}
	if !hasDecisionD1 {
		t.Errorf("Architectural Consistency Failed: previous decision D-001 not recalled")
	}

	// Repeated Mistakes Prevention: Must surface Failure F-001 (negative knowledge)
	hasFailureF1 := false
	for _, f := range pkg.PreviousFailures {
		if f.ID == "F-001" {
			hasFailureF1 = true
			break
		}
	}
	if !hasFailureF1 {
		t.Errorf("Repeated Mistakes Check Failed: failure F-001 not recalled")
	}

	// Cross-Task Experience Transfer: Must surface Lesson L-017 learned in Task B.2!
	hasLessonL17 := false
	for _, l := range pkg.RelevantLessons {
		if l.ID == "L-017" {
			hasLessonL17 = true
			if l.SourceTask != "B.2" {
				t.Errorf("Lesson L-017 source task mismatch: expected B.2, got %s", l.SourceTask)
			}
			if l.VerificationStatus != "VERIFIED" {
				t.Errorf("Lesson L-017 should be VERIFIED, got %s", l.VerificationStatus)
			}
			break
		}
	}
	if !hasLessonL17 {
		t.Errorf("Cross-Task Experience Transfer Failed: lesson L-017 from Task B.2 was not surfaced to Task A.3")
	}

	// Token & Context Budget Efficiency
	textContext := pkg.FormatHuman()
	tokens := context.EstimateTokens(textContext)
	if tokens > 1500 {
		t.Errorf("Context budget violated: expected <= 1500 tokens, got %d", tokens)
	}
	if tokens < 50 {
		t.Errorf("Context package appears too small: %d tokens", tokens)
	}

	t.Logf("EXPERIMENT RESULTS:")
	t.Logf("  Constraint Preservation: 100%% (Inherited %d active constraints)", len(pkg.AcceptedConstraints))
	t.Logf("  Architectural Consistency: Recalled Decision %s", decD1.Frontmatter.ID)
	t.Logf("  Mistake Avoidance: Recalled Known Failure %s", failF1.ID)
	t.Logf("  Cross-Task Transfer: Recalled Lesson %s from Task %s", lesL17.Frontmatter.ID, lesL17.Frontmatter.SourceTaskID)
	t.Logf("  Context Density: %d estimated tokens (Budget: 1500)", tokens)
	t.Logf("  Rehydrated Output:\n%s", textContext)
}
