package model

import "time"

// NodeType specifies the functional class of an entity in the Knodexa graph.
type NodeType string

const (
	NodeTypeProject     NodeType = "PROJECT"
	NodeTypeTask        NodeType = "TASK"
	NodeTypeDecision    NodeType = "DECISION"
	NodeTypeLesson      NodeType = "LESSON"
	NodeTypeObservation NodeType = "OBSERVATION"
	NodeTypeFailure     NodeType = "FAILURE"
	NodeTypeEvidence    NodeType = "EVIDENCE"
	NodeTypeRule        NodeType = "RULE"
	NodeTypeSkill       NodeType = "SKILL"
	NodeTypeFile        NodeType = "FILE"
	NodeTypeCommit      NodeType = "COMMIT"
)

// IsValid checks if the node type is supported.
func (nt NodeType) IsValid() bool {
	switch nt {
	case NodeTypeProject, NodeTypeTask, NodeTypeDecision, NodeTypeLesson,
		NodeTypeObservation, NodeTypeFailure, NodeTypeEvidence, NodeTypeRule,
		NodeTypeSkill, NodeTypeFile, NodeTypeCommit:
		return true
	default:
		return false
	}
}

// Node represents a generic indexed entity in Knodexa's SQLite and graph engine.
type Node struct {
	ID                 string                 `json:"id" yaml:"id"`
	Type               NodeType               `json:"type" yaml:"type"`
	ParentID           string                 `json:"parent_id,omitempty" yaml:"parent_id,omitempty"`
	Title              string                 `json:"title" yaml:"title"`
	Description        string                 `json:"description,omitempty" yaml:"description,omitempty"`
	Status             string                 `json:"status" yaml:"status"`
	Confidence         float64                `json:"confidence" yaml:"confidence"`
	VerificationStatus KnowledgeStatus        `json:"verification_status" yaml:"verification_status"`
	Source             string                 `json:"source,omitempty" yaml:"source,omitempty"`
	CreatedAt          time.Time              `json:"created_at" yaml:"created_at"`
	UpdatedAt          time.Time              `json:"updated_at" yaml:"updated_at"`
	FilePath           string                 `json:"file_path,omitempty" yaml:"file_path,omitempty"`
	ContentHash        string                 `json:"content_hash,omitempty" yaml:"content_hash,omitempty"`
	Tags               []string               `json:"tags,omitempty" yaml:"tags,omitempty"`
	TouchesFiles       []string               `json:"touches_files,omitempty" yaml:"touches_files,omitempty"`
	EvidenceIDs        []string               `json:"evidence_ids,omitempty" yaml:"evidence_ids,omitempty"`
	Metadata           map[string]interface{} `json:"metadata,omitempty" yaml:"metadata,omitempty"`
}

// ProjectConfig represents the root project manifest stored in .knodexa/project.yaml.
type ProjectConfig struct {
	ID                string    `yaml:"id" json:"id"`
	Name              string    `yaml:"name" json:"name"`
	Description       string    `yaml:"description,omitempty" json:"description,omitempty"`
	Status            string    `yaml:"status" json:"status"`
	CreatedAt         time.Time `yaml:"created_at" json:"created_at"`
	UpdatedAt         time.Time `yaml:"updated_at" json:"updated_at"`
	GlobalConstraints []string  `yaml:"global_constraints,omitempty" json:"global_constraints,omitempty"`
	RulesFiles        []string  `yaml:"rules_files,omitempty" json:"rules_files,omitempty"`
}

// TaskData represents a task or subtask stored in .knodexa/tasks/<dir>/<id>.yaml.
type TaskData struct {
	ID                   string     `yaml:"id" json:"id"`
	ParentID             string     `yaml:"parent_id,omitempty" json:"parent_id,omitempty"`
	Title                string     `yaml:"title" json:"title"`
	Goal                 string     `yaml:"goal" json:"goal"`
	Status               TaskStatus `yaml:"status" json:"status"`
	CreatedAt            time.Time  `yaml:"created_at" json:"created_at"`
	UpdatedAt            time.Time  `yaml:"updated_at" json:"updated_at"`
	Constraints          []string   `yaml:"constraints,omitempty" json:"constraints,omitempty"`
	TouchesFiles         []string   `yaml:"touches_files,omitempty" json:"touches_files,omitempty"`
	BaselineCommit       string     `yaml:"baseline_commit,omitempty" json:"baseline_commit,omitempty"`
	AcceptedCommit       string     `yaml:"accepted_commit,omitempty" json:"accepted_commit,omitempty"`
	RequiredVerification []string   `yaml:"required_verification,omitempty" json:"required_verification,omitempty"`
}

// DecisionFrontmatter represents the YAML frontmatter of an ADR in .knodexa/decisions/D-*.md.
type DecisionFrontmatter struct {
	ID           string          `yaml:"id" json:"id"`
	Title        string          `yaml:"title" json:"title"`
	Status       KnowledgeStatus `yaml:"status" json:"status"`
	TaskID       string          `yaml:"task_id,omitempty" json:"task_id,omitempty"`
	CreatedAt    time.Time       `yaml:"created_at" json:"created_at"`
	UpdatedAt    time.Time       `yaml:"updated_at" json:"updated_at"`
	Confidence   float64         `yaml:"confidence" json:"confidence"`
	Supersedes   string          `yaml:"supersedes,omitempty" json:"supersedes,omitempty"`
	SupersededBy string          `yaml:"superseded_by,omitempty" json:"superseded_by,omitempty"`
	EvidenceIDs  []string        `yaml:"evidence_ids,omitempty" json:"evidence_ids,omitempty"`
}

// DecisionDocument pairs the parsed frontmatter with the markdown body.
type DecisionDocument struct {
	Frontmatter DecisionFrontmatter `json:"frontmatter"`
	Body        string              `json:"body"`
}

// LessonFrontmatter represents the YAML frontmatter of a lesson in .knodexa/lessons/L-*.md.
type LessonFrontmatter struct {
	ID             string          `yaml:"id" json:"id"`
	Title          string          `yaml:"title" json:"title"`
	Status         KnowledgeStatus `yaml:"status" json:"status"`
	SourceTaskID   string          `yaml:"source_task_id,omitempty" json:"source_task_id,omitempty"`
	CreatedAt      time.Time       `yaml:"created_at" json:"created_at"`
	UpdatedAt      time.Time       `yaml:"updated_at" json:"updated_at"`
	Confidence     float64         `yaml:"confidence" json:"confidence"`
	Applicability  string          `yaml:"applicability" json:"applicability"`
	Tags           []string        `yaml:"tags,omitempty" json:"tags,omitempty"`
	RelatedFiles   []string        `yaml:"related_files,omitempty" json:"related_files,omitempty"`
	EvidenceIDs    []string        `yaml:"evidence_ids,omitempty" json:"evidence_ids,omitempty"`
}

// LessonDocument pairs the lesson frontmatter with the explanation body.
type LessonDocument struct {
	Frontmatter LessonFrontmatter `json:"frontmatter"`
	Body        string            `json:"body"`
}

// EvidenceData represents verification proof in .knodexa/evidence/E-*.yaml.
type EvidenceData struct {
	ID            string       `yaml:"id" json:"id"`
	Title         string       `yaml:"title" json:"title"`
	NodeID        string       `yaml:"node_id" json:"node_id"`
	EvidenceType  EvidenceType `yaml:"evidence_type" json:"evidence_type"`
	CreatedAt     time.Time    `yaml:"created_at" json:"created_at"`
	Command       string       `yaml:"command,omitempty" json:"command,omitempty"`
	ExitCode      int          `yaml:"exit_code" json:"exit_code"`
	OutputSummary string       `yaml:"output_summary,omitempty" json:"output_summary,omitempty"`
	CommitSHA     string       `yaml:"commit_sha,omitempty" json:"commit_sha,omitempty"`
	VerifiedBy    string       `yaml:"verified_by,omitempty" json:"verified_by,omitempty"`
}

// ObservationData represents an unverified observation in .knodexa/observations/OBS-*.yaml.
type ObservationData struct {
	ID        string    `yaml:"id" json:"id"`
	TaskID    string    `yaml:"task_id" json:"task_id"`
	Content   string    `yaml:"content" json:"content"`
	Files     []string  `yaml:"files,omitempty" json:"files,omitempty"`
	CreatedAt time.Time `yaml:"created_at" json:"created_at"`
}

// FailureData represents a recorded failed approach in .knodexa/failures/F-*.yaml.
type FailureData struct {
	ID            string    `yaml:"id" json:"id"`
	TaskID        string    `yaml:"task_id" json:"task_id"`
	Summary       string    `yaml:"summary" json:"summary"`
	CausalDetails string    `yaml:"causal_details" json:"causal_details"`
	AttemptedAt   time.Time `yaml:"attempted_at" json:"attempted_at"`
	EvidenceIDs   []string  `yaml:"evidence_ids,omitempty" json:"evidence_ids,omitempty"`
}
