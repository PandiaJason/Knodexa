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

func TestCanonicalTaskAndActionableContext(t *testing.T) {
	root := t.TempDir()
	store := storage.NewStorage(root)
	if _, err := store.Init("Context", "Regression"); err != nil {
		t.Fatal(err)
	}
	db, err := index.Open(filepath.Join(store.KnodexaDir(), "cache.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	idx := index.NewIndexer(db, store)
	for _, task := range []*model.TaskData{{ID: "root"}, {ID: "child", ParentID: "root"}, {ID: "leaf", ParentID: "child", TouchesFiles: []string{"auth.go"}, RequiredVerification: []string{"custom benchmark"}, BaselineCommit: "abc1234"}} {
		if err := store.SaveTask(task); err != nil {
			t.Fatal(err)
		}
	}
	if err := store.SaveDecision(&model.DecisionDocument{Frontmatter: model.DecisionFrontmatter{ID: "D-root", TaskID: "root", Status: model.StatusVerified}, Body: "Use immutable snapshots"}); err != nil {
		t.Fatal(err)
	}
	if err := store.SaveFailure(&model.FailureData{ID: "F-root", TaskID: "root", Summary: "Avoid global cache"}); err != nil {
		t.Fatal(err)
	}
	for _, lesson := range []*model.LessonDocument{
		{Frontmatter: model.LessonFrontmatter{ID: "L-good", SourceTaskID: "root", Status: model.StatusVerified, RelatedFiles: []string{"auth.go"}, Applicability: "When authenticating"}, Body: "Validate token expiry"},
		{Frontmatter: model.LessonFrontmatter{ID: "L-noise", SourceTaskID: "root", Status: model.StatusVerified, RelatedFiles: []string{"image.png"}, Confidence: 1}, Body: "Compress PNG files"},
	} {
		if err := store.SaveLesson(lesson); err != nil {
			t.Fatal(err)
		}
	}
	if err := idx.Sync(); err != nil {
		t.Fatal(err)
	}
	builder := NewBuilder(db, store, git.NewInspector(root))
	pkg, err := builder.Build("leaf", 1500)
	if err != nil {
		t.Fatal(err)
	}
	if len(pkg.RequiredVerification) != 1 || pkg.RequiredVerification[0] != "custom benchmark" {
		t.Fatalf("criteria lost: %+v", pkg.RequiredVerification)
	}
	if pkg.Baseline.StoredCommit != "abc1234" || pkg.Baseline.Commit != "abc1234" {
		t.Fatalf("baseline lost: %+v", pkg.Baseline)
	}
	if len(pkg.PreviousDecisions) != 1 || strings.TrimSpace(pkg.PreviousDecisions[0].Rationale) != "Use immutable snapshots" {
		t.Fatalf("ancestor rationale lost: %+v", pkg.PreviousDecisions)
	}
	if len(pkg.PreviousFailures) != 1 || pkg.PreviousFailures[0].ID != "F-root" {
		t.Fatalf("ancestor failure lost: %+v", pkg.PreviousFailures)
	}
	if len(pkg.RelevantLessons) != 1 || pkg.RelevantLessons[0].ID != "L-good" || pkg.RelevantLessons[0].Relevance != "high" {
		t.Fatalf("bad relevance: %+v", pkg.RelevantLessons)
	}
	if !strings.Contains(pkg.FormatHuman(), "Validate token expiry") || !strings.Contains(pkg.FormatHuman(), "When authenticating") {
		t.Fatal("lesson is not actionable")
	}
	tiny, err := builder.Build("leaf", 1)
	if err != nil {
		t.Fatal(err)
	}
	if !tiny.BudgetExceeded || !tiny.ContextPruned || tiny.TokenBudget != 1 || tiny.EstimatedTokens <= 1 || len(tiny.RequiredVerification) != 1 {
		t.Fatalf("overflow not explicit: %+v", tiny)
	}
}

func TestCompactContextRetainsRulesAndExpandsOnDemand(t *testing.T) {
	store := storage.NewStorage(t.TempDir())
	if _, err := store.Init("Compact", ""); err != nil {
		t.Fatal(err)
	}
	db, err := index.Open(filepath.Join(store.KnodexaDir(), "cache.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := store.SaveTask(&model.TaskData{ID: "T", Constraints: []string{"Preserve bool API"}, RequiredVerification: []string{"Run boundary tests"}, TouchesFiles: []string{"auth.go"}}); err != nil {
		t.Fatal(err)
	}
	body := "## Rationale\nUse strict integer validation.\n\n" + strings.Repeat("Historical rationale and rejected experiments.\n\n", 80) + "## Constraints\n- Preserve seven-second skew\n"
	if err := store.SaveDecision(&model.DecisionDocument{Frontmatter: model.DecisionFrontmatter{ID: "D-compact", TaskID: "T", Title: "Validate timestamps", Summary: "Reject bool, string, and float timestamps", Status: model.StatusVerified, Constraints: []string{"Never mutate claims"}, EvidenceIDs: []string{"E-1"}}, Body: body}); err != nil {
		t.Fatal(err)
	}
	if err := store.SaveLesson(&model.LessonDocument{Frontmatter: model.LessonFrontmatter{ID: "L-compact", SourceTaskID: "other", Title: "Bool is int", Status: model.StatusVerified, Summary: "Reject bool explicitly", RelatedFiles: []string{"auth.go"}, Applicability: "When validating timestamps"}, Body: "Reject bool explicitly.\n\n" + strings.Repeat("Long historical explanation.\n\n", 40)}); err != nil {
		t.Fatal(err)
	}
	if err := index.NewIndexer(db, store).Sync(); err != nil {
		t.Fatal(err)
	}
	builder := NewBuilder(db, store, git.NewInspector(store.RootDir()))
	compact, err := builder.BuildCompact("T", 100000)
	if err != nil {
		t.Fatal(err)
	}
	full, err := builder.Build("T", 100000)
	if err != nil {
		t.Fatal(err)
	}
	if compact.PreviousDecisions[0].Rationale != "" || compact.PreviousDecisions[0].Summary != "Reject bool, string, and float timestamps" {
		t.Fatal("compact rule lost or rationale repeated")
	}
	for _, rule := range []string{"Preserve bool API", "Never mutate claims", "Preserve seven-second skew", "Run boundary tests", "Reject bool explicitly"} {
		if !strings.Contains(compact.FormatHuman(), rule) {
			t.Fatalf("compact context lost %s", rule)
		}
	}
	if len(compact.FormatHuman())*2 >= len(full.FormatHuman()) {
		t.Fatal("verbose fixture did not become materially smaller")
	}
	detail, err := builder.GetDetail("D-compact")
	if err != nil {
		t.Fatal(err)
	}
	doc := detail.(*model.DecisionDocument)
	if strings.TrimSpace(doc.Body) != strings.TrimSpace(body) || len(doc.Frontmatter.EvidenceIDs) != 1 {
		t.Fatal("on-demand expansion lost full rationale or evidence IDs")
	}
	t.Logf("Compact/full human-text estimates: %d/%d", compact.EstimatedTokens, full.EstimatedTokens)
}
