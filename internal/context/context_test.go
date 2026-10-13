package context

import (
	"path/filepath"
	"strings"
	"testing"
	"time"

	"knodexa/internal/git"
	"knodexa/internal/index"
	"knodexa/internal/model"
	"knodexa/internal/storage"
)

func TestContextRehydrationStructure(t *testing.T) {
	tmpDir := t.TempDir()
	store := storage.NewStorage(tmpDir)
	_, _ = store.Init("Knodexa Context Test", "Testing context builder")

	dbPath := filepath.Join(tmpDir, ".knodexa", "cache.db")
	db, err := index.Open(dbPath)
	if err != nil {
		t.Fatalf("failed to open db: %v", err)
	}
	defer db.Close()

	indexer := index.NewIndexer(db, store)
	gitInspector := git.NewInspector(tmpDir)

	now := time.Now().UTC()

	// Task A
	_ = store.SaveTask(&model.TaskData{
		ID:          "A",
		Title:       "Core Sorting",
		Goal:        "High performance sorting",
		Status:      model.TaskStatusActive,
		CreatedAt:   now,
		UpdatedAt:   now,
		Constraints: []string{"Preserve public API", "Preserve deterministic output"},
	})

	// Subtask A.2
	_ = store.SaveTask(&model.TaskData{
		ID:                   "A.2",
		ParentID:             "A",
		Title:                "Optimize Sort",
		Goal:                 "Improve sorting performance.",
		Status:               model.TaskStatusActive,
		CreatedAt:            now,
		UpdatedAt:            now,
		Constraints:          []string{"Existing correctness suite must pass."},
		TouchesFiles:         []string{"internal/sort/sort.go"},
		BaselineCommit:       "abc1234",
		RequiredVerification: []string{"correctness tests", "regression tests", "performance benchmark"},
	})

	// Decision D-001
	_ = store.SaveDecision(&model.DecisionDocument{
		Frontmatter: model.DecisionFrontmatter{
			ID:          "D-001",
			Title:       "Benchmark before optimization.",
			Status:      model.StatusVerified,
			TaskID:      "A.2",
			CreatedAt:   now,
			UpdatedAt:   now,
			Confidence:  1.0,
			EvidenceIDs: []string{"E-001"},
		},
		Body: "Always profile before altering algorithms.",
	})

	// Failure F-002
	_ = store.SaveFailure(&model.FailureData{
		ID:            "F-002",
		TaskID:        "A.2",
		Summary:       "Parallel implementation regressed small inputs.",
		CausalDetails: "Goroutine overhead exceeded partition cost.",
		AttemptedAt:   now,
	})

	// Cross-task Lesson L-017 from Task B.2
	_ = store.SaveLesson(&model.LessonDocument{
		Frontmatter: model.LessonFrontmatter{
			ID:            "L-017",
			Title:         "Insertion sort threshold",
			Status:        model.StatusVerified,
			SourceTaskID:  "B.2",
			CreatedAt:     now,
			UpdatedAt:     now,
			Confidence:    0.95,
			Applicability: "When input length < 32",
			RelatedFiles:  []string{"internal/sort/sort.go"},
			EvidenceIDs:   []string{"E-002"},
		},
		Body: "Short-circuit small arrays to insertion sort.",
	})

	if err := indexer.Sync(); err != nil {
		t.Fatalf("indexer sync failed: %v", err)
	}

	builder := NewBuilder(db, store, gitInspector)
	pkg, err := builder.Build("A.2", 1500)
	if err != nil {
		t.Fatalf("build context failed: %v", err)
	}

	if pkg.Task.ID != "A.2" {
		t.Errorf("expected task A.2, got %s", pkg.Task.ID)
	}

	// Verify constraints inherited
	foundAPIConstraint := false
	for _, c := range pkg.AcceptedConstraints {
		if strings.Contains(c, "Preserve public API") {
			foundAPIConstraint = true
			break
		}
	}
	if !foundAPIConstraint {
		t.Errorf("expected inherited constraint 'Preserve public API', got %+v", pkg.AcceptedConstraints)
	}

	// Verify decisions
	if len(pkg.PreviousDecisions) == 0 || pkg.PreviousDecisions[0].ID != "D-001" {
		t.Errorf("expected D-001, got %+v", pkg.PreviousDecisions)
	}

	// Verify failures
	if len(pkg.PreviousFailures) == 0 || pkg.PreviousFailures[0].ID != "F-002" {
		t.Errorf("expected F-002, got %+v", pkg.PreviousFailures)
	}

	// Verify lesson L-017 from B.2
	if len(pkg.RelevantLessons) == 0 || pkg.RelevantLessons[0].ID != "L-017" {
		t.Errorf("expected L-017, got %+v", pkg.RelevantLessons)
	}

	humanOut := pkg.FormatHuman()
	if !strings.Contains(humanOut, "task: A.2") {
		t.Errorf("missing task in human output:\n%s", humanOut)
	}
	if !strings.Contains(humanOut, "D-001: Benchmark before optimization.") {
		t.Errorf("missing decision in human output:\n%s", humanOut)
	}
	if !strings.Contains(humanOut, "F-002: Parallel implementation regressed small inputs.") {
		t.Errorf("missing failure in human output:\n%s", humanOut)
	}
	if !strings.Contains(humanOut, "L-017 from Task B.2") {
		t.Errorf("missing lesson attribution in human output:\n%s", humanOut)
	}
}

func TestEstimateTokens(t *testing.T) {
	text := "This is a simple test string."
	tokens := EstimateTokens(text)
	if tokens <= 0 || tokens > len(text) {
		t.Errorf("unexpected token estimate: %d for %q", tokens, text)
	}
}
