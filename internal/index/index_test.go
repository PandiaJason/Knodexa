package index

import (
	"path/filepath"
	"testing"
	"time"

	"knodexa/internal/model"
	"knodexa/internal/storage"
)

func TestIndexSyncAndGetNode(t *testing.T) {
	tmpDir := t.TempDir()
	store := storage.NewStorage(tmpDir)
	_, err := store.Init("Knodexa Demo", "Testing Indexer")
	if err != nil {
		t.Fatalf("failed to init storage: %v", err)
	}

	dbPath := filepath.Join(tmpDir, ".knodexa", "cache.db")
	db, err := Open(dbPath)
	if err != nil {
		t.Fatalf("failed to open sqlite: %v", err)
	}
	defer db.Close()

	indexer := NewIndexer(db, store)
	if err := indexer.Sync(); err != nil {
		t.Fatalf("indexer sync failed: %v", err)
	}

	// Verify project node
	projNode, err := db.GetNode("proj_root")
	if err != nil {
		t.Fatalf("failed to get project node: %v", err)
	}
	if projNode.Title != "Knodexa Demo" || projNode.Type != model.NodeTypeProject {
		t.Errorf("unexpected project node: %+v", projNode)
	}

	// Add a task
	task := &model.TaskData{
		ID:          "A.1",
		ParentID:    "A",
		Title:       "Optimize Sort",
		Goal:        "Improve sorting speed",
		Status:      model.TaskStatusActive,
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
		Constraints: []string{"Preserve public API"},
	}
	if err := store.SaveTask(task); err != nil {
		t.Fatalf("failed to save task: %v", err)
	}

	// Re-sync
	if err := indexer.Sync(); err != nil {
		t.Fatalf("sync after task failed: %v", err)
	}

	taskNode, err := db.GetNode("A.1")
	if err != nil {
		t.Fatalf("failed to get task node: %v", err)
	}
	if taskNode.Title != "Optimize Sort" || taskNode.ParentID != "A" {
		t.Errorf("unexpected task node: %+v", taskNode)
	}

	constraints, err := db.GetUpstreamConstraints("A.1")
	if err != nil {
		t.Fatalf("failed to get constraints: %v", err)
	}
	if len(constraints) == 0 {
		t.Errorf("expected constraints, got 0")
	}
}

func TestRelatedLessonsCrossTask(t *testing.T) {
	tmpDir := t.TempDir()
	store := storage.NewStorage(tmpDir)
	_, _ = store.Init("Knodexa Test", "Testing Lessons")

	dbPath := filepath.Join(tmpDir, ".knodexa", "cache.db")
	db, err := Open(dbPath)
	if err != nil {
		t.Fatalf("failed to open sqlite: %v", err)
	}
	defer db.Close()

	indexer := NewIndexer(db, store)

	// Task A.2
	taskA2 := &model.TaskData{
		ID:           "A.2",
		ParentID:     "A",
		Title:        "Sort Benchmarks",
		Goal:         "Optimize slice sorting",
		Status:       model.TaskStatusActive,
		CreatedAt:    time.Now().UTC(),
		UpdatedAt:    time.Now().UTC(),
		TouchesFiles: []string{"sort.go"},
	}
	_ = store.SaveTask(taskA2)

	// Task B.2 produces Lesson L-017 touching sort.go
	lesson := &model.LessonDocument{
		Frontmatter: model.LessonFrontmatter{
			ID:            "L-017",
			Title:         "Insertion sort threshold",
			Status:        model.StatusVerified,
			SourceTaskID:  "B.2",
			CreatedAt:     time.Now().UTC(),
			UpdatedAt:     time.Now().UTC(),
			Confidence:    0.95,
			Applicability: "When sorting slices with <= 32 items",
			RelatedFiles:  []string{"sort.go"},
		},
		Body: "Switch to insertion sort for short arrays.",
	}
	_ = store.SaveLesson(lesson)

	if err := indexer.Sync(); err != nil {
		t.Fatalf("sync failed: %v", err)
	}

	related, err := db.GetRelatedLessons("A.2", []string{"sort.go"}, nil, 5)
	if err != nil {
		t.Fatalf("failed to get related lessons: %v", err)
	}
	if len(related) == 0 {
		t.Fatalf("expected at least 1 related lesson, got 0")
	}
	if related[0].ID != "L-017" {
		t.Errorf("expected L-017, got %s", related[0].ID)
	}
}

func TestRebuildFidelity(t *testing.T) {
	tmpDir := t.TempDir()
	store := storage.NewStorage(tmpDir)
	_, _ = store.Init("Knodexa Rebuild Test", "Testing 100% rebuild")

	dbPath := filepath.Join(tmpDir, ".knodexa", "cache.db")
	db, err := Open(dbPath)
	if err != nil {
		t.Fatalf("failed to open sqlite: %v", err)
	}
	defer db.Close()

	indexer := NewIndexer(db, store)

	// Add 2 tasks, 1 decision, 1 lesson, 1 evidence, 1 link
	now := time.Now().UTC()
	_ = store.SaveTask(&model.TaskData{
		ID:        "T.1",
		Title:     "Task 1",
		Status:    model.TaskStatusCompleted,
		CreatedAt: now,
		UpdatedAt: now,
	})
	_ = store.SaveTask(&model.TaskData{
		ID:        "T.2",
		Title:     "Task 2",
		Status:    model.TaskStatusActive,
		CreatedAt: now,
		UpdatedAt: now,
	})
	_ = store.SaveDecision(&model.DecisionDocument{
		Frontmatter: model.DecisionFrontmatter{
			ID:        "D-101",
			Title:     "Use SQLite",
			Status:    model.StatusVerified,
			TaskID:    "T.1",
			CreatedAt: now,
			UpdatedAt: now,
		},
		Body: "Decision details...",
	})
	_ = store.SaveLesson(&model.LessonDocument{
		Frontmatter: model.LessonFrontmatter{
			ID:           "L-201",
			Title:        "Always reset buffers",
			Status:       model.StatusVerified,
			SourceTaskID: "T.1",
			CreatedAt:    now,
			UpdatedAt:    now,
		},
		Body: "Lesson details...",
	})
	_ = store.SaveEvidence(&model.EvidenceData{
		ID:           "E-301",
		Title:        "Tests pass",
		NodeID:       "D-101",
		EvidenceType: model.EvidenceTestRun,
		CreatedAt:    now,
		ExitCode:     0,
	})
	_ = store.AppendLink(&model.Edge{
		FromID:    "L-201",
		ToID:      "T.2",
		Type:      model.EdgeRelevantTo,
		CreatedAt: now,
	})

	if err := indexer.Sync(); err != nil {
		t.Fatalf("initial sync failed: %v", err)
	}

	// Verify pre-rebuild count
	tasksPre, _ := db.GetNodesByType(model.NodeTypeTask)
	decisionsPre, _ := db.GetNodesByType(model.NodeTypeDecision)
	edgesPre, _ := db.GetEdgesFrom("L-201")

	// Trigger full Rebuild
	if err := indexer.Rebuild(); err != nil {
		t.Fatalf("indexer rebuild failed: %v", err)
	}

	tasksPost, _ := db.GetNodesByType(model.NodeTypeTask)
	decisionsPost, _ := db.GetNodesByType(model.NodeTypeDecision)
	edgesPost, _ := db.GetEdgesFrom("L-201")

	if len(tasksPre) != len(tasksPost) || len(tasksPost) != 2 {
		t.Errorf("task count mismatch post-rebuild: pre=%d, post=%d", len(tasksPre), len(tasksPost))
	}
	if len(decisionsPre) != len(decisionsPost) || len(decisionsPost) != 1 {
		t.Errorf("decision count mismatch post-rebuild: pre=%d, post=%d", len(decisionsPre), len(decisionsPost))
	}
	if len(edgesPre) != len(edgesPost) {
		t.Errorf("edge count mismatch post-rebuild: pre=%d, post=%d", len(edgesPre), len(edgesPost))
	}
}
