package storage

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"knodexa/internal/model"
)

func TestSanitize(t *testing.T) {
	// Valid IDs
	validIDs := []string{"A", "A.1", "D-001", "L-002", "task_foo-bar"}
	for _, id := range validIDs {
		if err := ValidateID(id); err != nil {
			t.Errorf("expected valid ID for %q, got error: %v", id, err)
		}
	}

	// Invalid IDs
	invalidIDs := []string{"", " ", "..", "../foo", "A/B", "A\\B", "id with spaces", "semi;colon"}
	for _, id := range invalidIDs {
		if err := ValidateID(id); err == nil {
			t.Errorf("expected error for invalid ID %q, got nil", id)
		}
	}

	// Sanitize paths
	validPaths := []string{"tasks/A/task.yaml", "decisions/D-001.md", "internal/sort/sort.go"}
	for _, p := range validPaths {
		if _, err := SanitizeRelPath(p); err != nil {
			t.Errorf("expected valid path %q, got: %v", p, err)
		}
	}

	invalidPaths := []string{"../outside", "tasks/../../etc/passwd", "/absolute/path"}
	for _, p := range invalidPaths {
		if _, err := SanitizeRelPath(p); err == nil {
			t.Errorf("expected error for path %q, got nil", p)
		}
	}
}

func TestStorageInitAndProject(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewStorage(tmpDir)

	if store.IsInitialized() {
		t.Errorf("store should not be initialized yet")
	}

	cfg, err := store.Init("Test Project", "Testing Knodexa storage")
	if err != nil {
		t.Fatalf("failed to init storage: %v", err)
	}

	if cfg.Name != "Test Project" {
		t.Errorf("expected project name 'Test Project', got %q", cfg.Name)
	}

	if !store.IsInitialized() {
		t.Errorf("store should now be initialized")
	}

	readCfg, err := store.ReadProject()
	if err != nil {
		t.Fatalf("failed to read project: %v", err)
	}
	if readCfg.Name != "Test Project" || readCfg.ID != "proj_root" {
		t.Errorf("unexpected project data: %+v", readCfg)
	}
}

func TestStorageTaskRoundTrip(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewStorage(tmpDir)
	_, _ = store.Init("Test", "Desc")

	task := &model.TaskData{
		ID:                   "A.1",
		ParentID:             "A",
		Title:                "Subtask A.1",
		Goal:                 "Benchmark sorting algorithm",
		Status:               model.TaskStatusActive,
		CreatedAt:            time.Now().UTC().Truncate(time.Second),
		UpdatedAt:            time.Now().UTC().Truncate(time.Second),
		Constraints:          []string{"Must pass test suite", "Public API unchanged"},
		TouchesFiles:         []string{"sort.go", "sort_test.go"},
		BaselineCommit:       "abc1234",
		RequiredVerification: []string{"go test ./..."},
	}

	if err := store.SaveTask(task); err != nil {
		t.Fatalf("failed to save task: %v", err)
	}

	readTask, err := store.ReadTask("A.1")
	if err != nil {
		t.Fatalf("failed to read task: %v", err)
	}

	if readTask.ID != "A.1" || readTask.ParentID != "A" || readTask.Goal != task.Goal {
		t.Errorf("task mismatch: expected %+v, got %+v", task, readTask)
	}

	tasks, err := store.ListTasks()
	if err != nil {
		t.Fatalf("failed to list tasks: %v", err)
	}
	if len(tasks) != 1 {
		t.Fatalf("expected 1 task, got %d", len(tasks))
	}
	if tasks[0].ID != "A.1" {
		t.Errorf("expected task A.1, got %s", tasks[0].ID)
	}
}

func TestStorageDecisionMarkdown(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewStorage(tmpDir)
	_, _ = store.Init("Test", "Desc")

	now := time.Now().UTC().Truncate(time.Second)
	doc := &model.DecisionDocument{
		Frontmatter: model.DecisionFrontmatter{
			ID:          "D-001",
			Title:       "Adopt SQLite",
			Status:      model.StatusVerified,
			TaskID:      "A.1",
			CreatedAt:   now,
			UpdatedAt:   now,
			Confidence:  0.95,
			EvidenceIDs: []string{"E-001"},
		},
		Body: "## Context\nWe need embedded fast indexing.\n\n## Decision\nUse SQLite pure-Go.",
	}

	if err := store.SaveDecision(doc); err != nil {
		t.Fatalf("failed to save decision: %v", err)
	}

	readDoc, err := store.ReadDecision("D-001")
	if err != nil {
		t.Fatalf("failed to read decision: %v", err)
	}

	if readDoc.Frontmatter.ID != "D-001" || readDoc.Frontmatter.Title != "Adopt SQLite" {
		t.Errorf("frontmatter mismatch: %+v", readDoc.Frontmatter)
	}
	if len(readDoc.Frontmatter.EvidenceIDs) != 1 || readDoc.Frontmatter.EvidenceIDs[0] != "E-001" {
		t.Errorf("evidence IDs mismatch: %+v", readDoc.Frontmatter.EvidenceIDs)
	}

	decisions, err := store.ListDecisions()
	if err != nil {
		t.Fatalf("failed to list decisions: %v", err)
	}
	if len(decisions) != 1 {
		t.Fatalf("expected 1 decision, got %d", len(decisions))
	}
}

func TestStorageLessonMarkdown(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewStorage(tmpDir)
	_, _ = store.Init("Test", "Desc")

	now := time.Now().UTC().Truncate(time.Second)
	doc := &model.LessonDocument{
		Frontmatter: model.LessonFrontmatter{
			ID:            "L-017",
			Title:         "Reset pool buffers",
			Status:        model.StatusVerified,
			SourceTaskID:  "B.2",
			CreatedAt:     now,
			UpdatedAt:     now,
			Confidence:    0.9,
			Applicability: "When reusing buffers in sync.Pool",
			Tags:          []string{"memory", "concurrency"},
			RelatedFiles:  []string{"pool.go"},
			EvidenceIDs:   []string{"E-002"},
		},
		Body: "Always call `buf.Reset()` before putting buffer back in pool.",
	}

	if err := store.SaveLesson(doc); err != nil {
		t.Fatalf("failed to save lesson: %v", err)
	}

	readDoc, err := store.ReadLesson("L-017")
	if err != nil {
		t.Fatalf("failed to read lesson: %v", err)
	}

	if readDoc.Frontmatter.ID != "L-017" || readDoc.Frontmatter.Applicability != doc.Frontmatter.Applicability {
		t.Errorf("lesson mismatch: %+v", readDoc.Frontmatter)
	}

	lessons, err := store.ListLessons()
	if err != nil {
		t.Fatalf("failed to list lessons: %v", err)
	}
	if len(lessons) != 1 {
		t.Fatalf("expected 1 lesson, got %d", len(lessons))
	}
}

func TestStorageEvidenceAndLedgers(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewStorage(tmpDir)
	_, _ = store.Init("Test", "Desc")

	now := time.Now().UTC().Truncate(time.Second)
	ev := &model.EvidenceData{
		ID:            "E-001",
		Title:         "Correctness tests passed",
		NodeID:        "D-001",
		EvidenceType:  model.EvidenceTestRun,
		CreatedAt:     now,
		Command:       "go test ./...",
		ExitCode:      0,
		OutputSummary: "PASS: all tests green",
		CommitSHA:     "c0ffee",
		VerifiedBy:    "agent",
	}

	if err := store.SaveEvidence(ev); err != nil {
		t.Fatalf("failed to save evidence: %v", err)
	}

	readEv, err := store.ReadEvidence("E-001")
	if err != nil {
		t.Fatalf("failed to read evidence: %v", err)
	}
	if readEv.ID != "E-001" || readEv.ExitCode != 0 {
		t.Errorf("evidence mismatch: %+v", readEv)
	}

	// Links
	edge := &model.Edge{
		FromID:    "A.1",
		ToID:      "D-001",
		Type:      model.EdgeProduced,
		CreatedAt: now,
	}
	if err := store.AppendLink(edge); err != nil {
		t.Fatalf("failed to append link: %v", err)
	}

	links, err := store.ReadLinks()
	if err != nil {
		t.Fatalf("failed to read links: %v", err)
	}
	if len(links) != 1 || links[0].FromID != "A.1" || links[0].ToID != "D-001" {
		t.Errorf("unexpected links: %+v", links)
	}

	// Events
	evt := &model.Event{
		EventID:   "evt-1",
		Timestamp: now,
		Actor:     "agent",
		Action:    "CREATE_TASK",
		TargetID:  "A.1",
	}
	if err := store.AppendEvent(evt); err != nil {
		t.Fatalf("failed to append event: %v", err)
	}

	events, err := store.ReadEvents()
	if err != nil {
		t.Fatalf("failed to read events: %v", err)
	}
	if len(events) != 1 || events[0].EventID != "evt-1" {
		t.Errorf("unexpected events: %+v", events)
	}
}

func TestAtomicWriteFileIntegrity(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewStorage(tmpDir)
	_, _ = store.Init("Test", "Desc")

	target := filepath.Join(store.KnodexaDir(), "test_atomic.txt")
	err := store.atomicWriteFile(target, []byte("hello world"), 0644)
	if err != nil {
		t.Fatalf("atomic write failed: %v", err)
	}

	content, err := os.ReadFile(target)
	if err != nil {
		t.Fatalf("failed to read atomic file: %v", err)
	}
	if string(content) != "hello world" {
		t.Errorf("expected 'hello world', got %q", string(content))
	}
}
