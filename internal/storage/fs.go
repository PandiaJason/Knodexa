package storage

import (
	"bufio"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"gopkg.in/yaml.v3"
	"knodexa/internal/model"
)

const KnodexaDir = ".knodexa"

var ErrConflict = errors.New("task changed since it was read; reload and retry")
var ErrTaskNotFound = errors.New("task not found")

// Storage manages canonical file I/O within the repository's .knodexa directory.
type Storage struct {
	rootDir    string
	knodexaDir string
	mu         sync.RWMutex
}

// NewStorage constructs a Storage instance for a repository root.
func NewStorage(rootDir string) *Storage {
	absRoot, err := filepath.Abs(rootDir)
	if err != nil {
		absRoot = rootDir
	}
	return &Storage{
		rootDir:    absRoot,
		knodexaDir: filepath.Join(absRoot, KnodexaDir),
	}
}

// RootDir returns the project root directory.
func (s *Storage) RootDir() string {
	return s.rootDir
}

// KnodexaDir returns the .knodexa directory path.
func (s *Storage) KnodexaDir() string {
	return s.knodexaDir
}

// IsInitialized checks if the .knodexa directory and project.yaml exist.
func (s *Storage) IsInitialized() bool {
	projPath := filepath.Join(s.knodexaDir, "project.yaml")
	info, err := os.Stat(projPath)
	return err == nil && !info.IsDir()
}

// Init creates the directory tree and initial project.yaml.
func (s *Storage) Init(name, description string) (*model.ProjectConfig, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	dirs := []string{
		s.knodexaDir,
		filepath.Join(s.knodexaDir, "tasks"),
		filepath.Join(s.knodexaDir, "decisions"),
		filepath.Join(s.knodexaDir, "lessons"),
		filepath.Join(s.knodexaDir, "evidence"),
		filepath.Join(s.knodexaDir, "observations"),
		filepath.Join(s.knodexaDir, "failures"),
	}

	for _, d := range dirs {
		if err := os.MkdirAll(d, 0755); err != nil {
			return nil, fmt.Errorf("failed to create directory %s: %w", d, err)
		}
	}

	now := time.Now().UTC()
	cfg := &model.ProjectConfig{
		ID:          "proj_root",
		Name:        name,
		Description: description,
		Status:      "ACTIVE",
		CreatedAt:   now,
		UpdatedAt:   now,
		GlobalConstraints: []string{
			"Deterministic behavior: zero LLM calls in core engine",
			"Evidence-driven verification for decisions and lessons",
			"Preserve existing repository constraints and public APIs",
		},
		RulesFiles: []string{"AGENTS.md"},
	}

	projBytes, err := yaml.Marshal(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal project.yaml: %w", err)
	}

	projPath := filepath.Join(s.knodexaDir, "project.yaml")
	if err := s.atomicWriteFile(projPath, projBytes, 0644); err != nil {
		return nil, fmt.Errorf("failed to write project.yaml: %w", err)
	}

	// Ensure events.jsonl and links.jsonl exist
	for _, f := range []string{"events.jsonl", "links.jsonl"} {
		p := filepath.Join(s.knodexaDir, f)
		if _, err := os.Stat(p); os.IsNotExist(err) {
			if err := os.WriteFile(p, []byte(""), 0644); err != nil {
				return nil, fmt.Errorf("failed to create %s: %w", f, err)
			}
		}
	}

	return cfg, nil
}

// ReadProject reads the project configuration from .knodexa/project.yaml.
func (s *Storage) ReadProject() (*model.ProjectConfig, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	projPath := filepath.Join(s.knodexaDir, "project.yaml")
	data, err := os.ReadFile(projPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read %s: %w", projPath, err)
	}

	var cfg model.ProjectConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal project.yaml: %w", err)
	}
	return &cfg, nil
}

// SaveTask writes a task YAML to .knodexa/tasks/<dir>/<id>.yaml.
func (s *Storage) SaveTask(task *model.TaskData) error {
	if err := ValidateID(task.ID); err != nil {
		return fmt.Errorf("invalid task id: %w", err)
	}
	return WithProcessLock(filepath.Join(s.knodexaDir, "tasks.lock"), func() error {
		s.mu.Lock()
		defer s.mu.Unlock()
		return s.saveTask(task)
	})
}

// UpdateTask applies mergeable changes to freshly read canonical data while the
// process lock is held. Callbacks must not invoke other mutation operations.
func (s *Storage) UpdateTask(id string, change func(*model.TaskData) error) error {
	if err := ValidateID(id); err != nil {
		return err
	}
	return WithProcessLock(filepath.Join(s.knodexaDir, "tasks.lock"), func() error {
		s.mu.Lock()
		defer s.mu.Unlock()
		task, err := s.readTask(id)
		if err != nil {
			return err
		}
		before, err := yaml.Marshal(task)
		if err != nil {
			return err
		}
		if err := change(task); err != nil {
			return err
		}
		if task.ID != id {
			return fmt.Errorf("UpdateTask cannot change task identity")
		}
		after, err := yaml.Marshal(task)
		if err != nil {
			return err
		}
		if string(before) == string(after) {
			return nil
		}
		return s.saveTask(task)
	})
}

func (s *Storage) saveTask(task *model.TaskData) error {
	current, err := s.readTask(task.ID)
	if err != nil && !errors.Is(err, ErrTaskNotFound) {
		return err
	}
	if current != nil {
		if task.SnapshotHash == "" || task.SnapshotHash != current.SnapshotHash {
			return fmt.Errorf("%w: %s", ErrConflict, task.ID)
		}
	}

	var dir string
	if task.ParentID != "" {
		if err := ValidateID(task.ParentID); err != nil {
			return fmt.Errorf("invalid parent task id: %w", err)
		}
		dir = filepath.Join(s.knodexaDir, "tasks", task.ParentID)
	} else {
		dir = filepath.Join(s.knodexaDir, "tasks", task.ID)
	}

	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create task dir: %w", err)
	}

	filePath := filepath.Join(dir, fmt.Sprintf("%s.yaml", task.ID))
	data, err := yaml.Marshal(task)
	if err != nil {
		return fmt.Errorf("failed to marshal task yaml: %w", err)
	}

	if err := s.atomicWriteFile(filePath, data, 0644); err != nil {
		return err
	}
	if current != nil && current.SnapshotPath != filePath {
		if err := os.Remove(current.SnapshotPath); err != nil {
			return fmt.Errorf("remove previous task location: %w", err)
		}
	}
	hash := sha256.Sum256(data)
	task.SnapshotHash = hex.EncodeToString(hash[:])
	task.SnapshotPath = filePath
	return nil
}

// ReadTask loads a task YAML by ID.
func (s *Storage) ReadTask(taskID string) (*model.TaskData, error) {
	if err := ValidateID(taskID); err != nil {
		return nil, fmt.Errorf("invalid task id: %w", err)
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.readTask(taskID)
}

func (s *Storage) readTask(taskID string) (*model.TaskData, error) {

	tasksDir := filepath.Join(s.knodexaDir, "tasks")
	var foundPath string

	err := filepath.Walk(tasksDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		if info.Name() == fmt.Sprintf("%s.yaml", taskID) {
			foundPath = path
			return filepath.SkipAll
		}
		return nil
	})
	if err != nil && err != filepath.SkipAll {
		return nil, err
	}
	if foundPath == "" {
		return nil, fmt.Errorf("%w: %s", ErrTaskNotFound, taskID)
	}

	data, err := os.ReadFile(foundPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read task file: %w", err)
	}

	var task model.TaskData
	if err := yaml.Unmarshal(data, &task); err != nil {
		return nil, fmt.Errorf("failed to unmarshal task %s: %w", taskID, err)
	}
	hash := sha256.Sum256(data)
	task.SnapshotHash = hex.EncodeToString(hash[:])
	task.SnapshotPath = foundPath
	return &task, nil
}

// ListTasks recursively finds all tasks in .knodexa/tasks.
func (s *Storage) ListTasks() ([]model.TaskData, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	tasksDir := filepath.Join(s.knodexaDir, "tasks")
	var tasks []model.TaskData

	if _, err := os.Stat(tasksDir); os.IsNotExist(err) {
		return tasks, nil
	}

	err := filepath.Walk(tasksDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() || !strings.HasSuffix(info.Name(), ".yaml") {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("read %s: %w", path, err)
		}
		var task model.TaskData
		if err := yaml.Unmarshal(data, &task); err != nil {
			return fmt.Errorf("parse %s: %w", path, err)
		}
		if err := ValidateID(task.ID); err != nil {
			return fmt.Errorf("%s: %w", path, err)
		}
		hash := sha256.Sum256(data)
		task.SnapshotHash = hex.EncodeToString(hash[:])
		task.SnapshotPath = path
		tasks = append(tasks, task)
		return nil
	})
	if err != nil {
		return nil, err
	}

	sort.Slice(tasks, func(i, j int) bool {
		return tasks[i].ID < tasks[j].ID
	})
	return tasks, nil
}

// SaveDecision writes an ADR markdown file with frontmatter to .knodexa/decisions/D-<id>.md.
func (s *Storage) SaveDecision(doc *model.DecisionDocument) error {
	if err := ValidateID(doc.Frontmatter.ID); err != nil {
		return fmt.Errorf("invalid decision id: %w", err)
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	content, err := RenderWithFrontmatter(doc.Frontmatter, doc.Body)
	if err != nil {
		return fmt.Errorf("failed to format decision markdown: %w", err)
	}

	dest := filepath.Join(s.knodexaDir, "decisions", fmt.Sprintf("%s.md", doc.Frontmatter.ID))
	return s.atomicWriteFile(dest, content, 0644)
}

// ReadDecision reads an ADR markdown file by ID.
func (s *Storage) ReadDecision(id string) (*model.DecisionDocument, error) {
	if err := ValidateID(id); err != nil {
		return nil, fmt.Errorf("invalid decision id: %w", err)
	}
	s.mu.RLock()
	defer s.mu.RUnlock()

	dest := filepath.Join(s.knodexaDir, "decisions", fmt.Sprintf("%s.md", id))
	data, err := os.ReadFile(dest)
	if err != nil {
		return nil, fmt.Errorf("failed to read decision %s: %w", id, err)
	}

	yamlBlock, body, err := SplitFrontmatter(data)
	if err != nil {
		return nil, fmt.Errorf("failed to parse decision %s frontmatter: %w", id, err)
	}

	var fm model.DecisionFrontmatter
	if err := yaml.Unmarshal(yamlBlock, &fm); err != nil {
		return nil, fmt.Errorf("failed to unmarshal decision frontmatter: %w", err)
	}

	return &model.DecisionDocument{
		Frontmatter: fm,
		Body:        body,
	}, nil
}

// ListDecisions loads all decisions from .knodexa/decisions.
func (s *Storage) ListDecisions() ([]model.DecisionDocument, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	dir := filepath.Join(s.knodexaDir, "decisions")
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var results []model.DecisionDocument
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(dir, entry.Name()))
		if err != nil {
			return nil, fmt.Errorf("read %s: %w", entry.Name(), err)
		}
		yamlBlock, body, err := SplitFrontmatter(data)
		if err != nil {
			return nil, fmt.Errorf("parse %s: %w", entry.Name(), err)
		}
		var fm model.DecisionFrontmatter
		if err := yaml.Unmarshal(yamlBlock, &fm); err != nil {
			return nil, fmt.Errorf("parse %s: %w", entry.Name(), err)
		}
		if err := ValidateID(fm.ID); err != nil {
			return nil, fmt.Errorf("%s: %w", entry.Name(), err)
		}
		results = append(results, model.DecisionDocument{Frontmatter: fm, Body: body})
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].Frontmatter.ID < results[j].Frontmatter.ID
	})
	return results, nil
}

// SaveLesson writes a lesson markdown file to .knodexa/lessons/L-<id>.md.
func (s *Storage) SaveLesson(doc *model.LessonDocument) error {
	if err := ValidateID(doc.Frontmatter.ID); err != nil {
		return fmt.Errorf("invalid lesson id: %w", err)
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	content, err := RenderWithFrontmatter(doc.Frontmatter, doc.Body)
	if err != nil {
		return fmt.Errorf("failed to format lesson markdown: %w", err)
	}

	dest := filepath.Join(s.knodexaDir, "lessons", fmt.Sprintf("%s.md", doc.Frontmatter.ID))
	return s.atomicWriteFile(dest, content, 0644)
}

// ReadLesson loads a lesson by ID.
func (s *Storage) ReadLesson(id string) (*model.LessonDocument, error) {
	if err := ValidateID(id); err != nil {
		return nil, fmt.Errorf("invalid lesson id: %w", err)
	}
	s.mu.RLock()
	defer s.mu.RUnlock()

	dest := filepath.Join(s.knodexaDir, "lessons", fmt.Sprintf("%s.md", id))
	data, err := os.ReadFile(dest)
	if err != nil {
		return nil, fmt.Errorf("failed to read lesson %s: %w", id, err)
	}

	yamlBlock, body, err := SplitFrontmatter(data)
	if err != nil {
		return nil, fmt.Errorf("failed to parse lesson %s frontmatter: %w", id, err)
	}

	var fm model.LessonFrontmatter
	if err := yaml.Unmarshal(yamlBlock, &fm); err != nil {
		return nil, fmt.Errorf("failed to unmarshal lesson frontmatter: %w", err)
	}

	return &model.LessonDocument{
		Frontmatter: fm,
		Body:        body,
	}, nil
}

// ListLessons loads all lessons from .knodexa/lessons.
func (s *Storage) ListLessons() ([]model.LessonDocument, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	dir := filepath.Join(s.knodexaDir, "lessons")
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var results []model.LessonDocument
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(dir, entry.Name()))
		if err != nil {
			return nil, fmt.Errorf("read %s: %w", entry.Name(), err)
		}
		yamlBlock, body, err := SplitFrontmatter(data)
		if err != nil {
			return nil, fmt.Errorf("parse %s: %w", entry.Name(), err)
		}
		var fm model.LessonFrontmatter
		if err := yaml.Unmarshal(yamlBlock, &fm); err != nil {
			return nil, fmt.Errorf("parse %s: %w", entry.Name(), err)
		}
		if err := ValidateID(fm.ID); err != nil {
			return nil, fmt.Errorf("%s: %w", entry.Name(), err)
		}
		results = append(results, model.LessonDocument{Frontmatter: fm, Body: body})
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].Frontmatter.ID < results[j].Frontmatter.ID
	})
	return results, nil
}

// SaveEvidence writes evidence YAML to .knodexa/evidence/E-<id>.yaml.
func (s *Storage) SaveEvidence(ev *model.EvidenceData) error {
	if err := ValidateID(ev.ID); err != nil {
		return fmt.Errorf("invalid evidence id: %w", err)
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := yaml.Marshal(ev)
	if err != nil {
		return fmt.Errorf("failed to marshal evidence: %w", err)
	}

	dest := filepath.Join(s.knodexaDir, "evidence", fmt.Sprintf("%s.yaml", ev.ID))
	return s.atomicWriteFile(dest, data, 0644)
}

// ReadEvidence loads evidence YAML by ID.
func (s *Storage) ReadEvidence(id string) (*model.EvidenceData, error) {
	if err := ValidateID(id); err != nil {
		return nil, fmt.Errorf("invalid evidence id: %w", err)
	}
	s.mu.RLock()
	defer s.mu.RUnlock()

	dest := filepath.Join(s.knodexaDir, "evidence", fmt.Sprintf("%s.yaml", id))
	data, err := os.ReadFile(dest)
	if err != nil {
		return nil, fmt.Errorf("failed to read evidence %s: %w", id, err)
	}

	var ev model.EvidenceData
	if err := yaml.Unmarshal(data, &ev); err != nil {
		return nil, fmt.Errorf("failed to unmarshal evidence: %w", err)
	}
	var execution struct {
		ExitCode *int `yaml:"exit_code"`
	}
	if err := yaml.Unmarshal(data, &execution); err != nil {
		return nil, err
	}
	ev.ExitCodeRecorded = execution.ExitCode != nil
	return &ev, nil
}

// ListEvidence loads all evidence items.
func (s *Storage) ListEvidence() ([]model.EvidenceData, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	dir := filepath.Join(s.knodexaDir, "evidence")
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var results []model.EvidenceData
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".yaml") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(dir, entry.Name()))
		if err != nil {
			return nil, fmt.Errorf("read %s: %w", entry.Name(), err)
		}
		var ev model.EvidenceData
		if err := yaml.Unmarshal(data, &ev); err != nil {
			return nil, fmt.Errorf("parse %s: %w", entry.Name(), err)
		}
		if err := ValidateID(ev.ID); err != nil {
			return nil, fmt.Errorf("%s: %w", entry.Name(), err)
		}
		results = append(results, ev)
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].ID < results[j].ID
	})
	return results, nil
}

// SaveObservation writes an observation YAML to .knodexa/observations/OBS-<id>.yaml.
func (s *Storage) SaveObservation(obs *model.ObservationData) error {
	if err := ValidateID(obs.ID); err != nil {
		return fmt.Errorf("invalid observation id: %w", err)
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := yaml.Marshal(obs)
	if err != nil {
		return fmt.Errorf("failed to marshal observation: %w", err)
	}

	dest := filepath.Join(s.knodexaDir, "observations", fmt.Sprintf("%s.yaml", obs.ID))
	return s.atomicWriteFile(dest, data, 0644)
}

// ListObservations loads all observations.
func (s *Storage) ListObservations() ([]model.ObservationData, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	dir := filepath.Join(s.knodexaDir, "observations")
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var results []model.ObservationData
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".yaml") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(dir, entry.Name()))
		if err != nil {
			continue
		}
		var obs model.ObservationData
		if err := yaml.Unmarshal(data, &obs); err == nil && obs.ID != "" {
			results = append(results, obs)
		}
	}
	return results, nil
}

// SaveFailure writes a failure YAML to .knodexa/failures/F-<id>.yaml.
func (s *Storage) SaveFailure(fail *model.FailureData) error {
	if err := ValidateID(fail.ID); err != nil {
		return fmt.Errorf("invalid failure id: %w", err)
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := yaml.Marshal(fail)
	if err != nil {
		return fmt.Errorf("failed to marshal failure: %w", err)
	}

	dest := filepath.Join(s.knodexaDir, "failures", fmt.Sprintf("%s.yaml", fail.ID))
	return s.atomicWriteFile(dest, data, 0644)
}

// ListFailures loads all failure records.
func (s *Storage) ListFailures() ([]model.FailureData, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	dir := filepath.Join(s.knodexaDir, "failures")
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var results []model.FailureData
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".yaml") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(dir, entry.Name()))
		if err != nil {
			return nil, fmt.Errorf("read %s: %w", entry.Name(), err)
		}
		var fail model.FailureData
		if err := yaml.Unmarshal(data, &fail); err != nil {
			return nil, fmt.Errorf("parse %s: %w", entry.Name(), err)
		}
		if err := ValidateID(fail.ID); err != nil {
			return nil, fmt.Errorf("%s: %w", entry.Name(), err)
		}
		results = append(results, fail)
	}
	return results, nil
}

// AppendLink appends an edge to .knodexa/links.jsonl.
func (s *Storage) AppendLink(edge *model.Edge) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	line, err := json.Marshal(edge)
	if err != nil {
		return fmt.Errorf("failed to marshal link: %w", err)
	}

	p := filepath.Join(s.knodexaDir, "links.jsonl")
	f, err := os.OpenFile(p, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open links.jsonl: %w", err)
	}
	defer f.Close()

	if _, err := f.Write(append(line, '\n')); err != nil {
		return fmt.Errorf("failed to write to links.jsonl: %w", err)
	}
	return nil
}

// ReadLinks reads all edges from .knodexa/links.jsonl.
func (s *Storage) ReadLinks() ([]model.Edge, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	p := filepath.Join(s.knodexaDir, "links.jsonl")
	f, err := os.Open(p)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	defer f.Close()

	var edges []model.Edge
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var edge model.Edge
		if err := json.Unmarshal([]byte(line), &edge); err != nil {
			return nil, fmt.Errorf("parse links.jsonl: %w", err)
		}
		if edge.FromID == "" || edge.ToID == "" {
			return nil, fmt.Errorf("links.jsonl: edge endpoints are required")
		}
		edges = append(edges, edge)
	}
	return edges, scanner.Err()
}

// AppendEvent appends an event to .knodexa/events.jsonl.
func (s *Storage) AppendEvent(evt *model.Event) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	line, err := json.Marshal(evt)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	p := filepath.Join(s.knodexaDir, "events.jsonl")
	f, err := os.OpenFile(p, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open events.jsonl: %w", err)
	}
	defer f.Close()

	if _, err := f.Write(append(line, '\n')); err != nil {
		return fmt.Errorf("failed to write to events.jsonl: %w", err)
	}
	return nil
}

// ReadEvents reads all events from .knodexa/events.jsonl.
func (s *Storage) ReadEvents() ([]model.Event, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	p := filepath.Join(s.knodexaDir, "events.jsonl")
	f, err := os.Open(p)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	defer f.Close()

	var events []model.Event
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var evt model.Event
		if err := json.Unmarshal([]byte(line), &evt); err == nil {
			events = append(events, evt)
		}
	}
	return events, scanner.Err()
}

// atomicWriteFile safely writes content to a temporary file in the same directory before renaming it.
func (s *Storage) atomicWriteFile(filePath string, data []byte, perm os.FileMode) error {
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	randBytes := make([]byte, 8)
	_, _ = rand.Read(randBytes)
	tmpPath := filepath.Join(dir, fmt.Sprintf(".tmp_%s_%s", filepath.Base(filePath), hex.EncodeToString(randBytes)))

	f, err := os.OpenFile(tmpPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, perm)
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}

	if _, err := f.Write(data); err != nil {
		f.Close()
		os.Remove(tmpPath)
		return fmt.Errorf("failed to write temp file: %w", err)
	}

	if err := f.Sync(); err != nil {
		f.Close()
		os.Remove(tmpPath)
		return fmt.Errorf("failed to sync temp file: %w", err)
	}

	if err := f.Close(); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("failed to close temp file: %w", err)
	}

	if err := os.Rename(tmpPath, filePath); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("failed to rename temp file to %s: %w", filePath, err)
	}

	return nil
}
