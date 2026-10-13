package model

import "time"

// EdgeType represents the typed relationship between two knowledge nodes.
type EdgeType string

const (
	EdgeParentOf      EdgeType = "PARENT_OF"
	EdgeDependsOn     EdgeType = "DEPENDS_ON"
	EdgeProduced      EdgeType = "PRODUCED"
	EdgeLearnedFrom   EdgeType = "LEARNED_FROM"
	EdgeRelevantTo    EdgeType = "RELEVANT_TO"
	EdgeSupersedes    EdgeType = "SUPERSEDES"
	EdgeVerifiedBy    EdgeType = "VERIFIED_BY"
	EdgeFailedBecause EdgeType = "FAILED_BECAUSE"
	EdgeImplementedBy EdgeType = "IMPLEMENTED_BY"
	EdgeTouchesFile   EdgeType = "TOUCHES_FILE"
	EdgeUsesSkill     EdgeType = "USES_SKILL"
	EdgeConstrainedBy EdgeType = "CONSTRAINED_BY"
)

// IsValid checks if the edge type is supported.
func (e EdgeType) IsValid() bool {
	switch e {
	case EdgeParentOf, EdgeDependsOn, EdgeProduced, EdgeLearnedFrom, EdgeRelevantTo,
		EdgeSupersedes, EdgeVerifiedBy, EdgeFailedBecause, EdgeImplementedBy,
		EdgeTouchesFile, EdgeUsesSkill, EdgeConstrainedBy:
		return true
	default:
		return false
	}
}

// Edge represents a directed, typed relationship between two nodes in the Knodexa graph.
type Edge struct {
	FromID    string    `json:"from_id" yaml:"from_id"`
	ToID      string    `json:"to_id" yaml:"to_id"`
	Type      EdgeType  `json:"type" yaml:"type"`
	CreatedAt time.Time `json:"created_at" yaml:"created_at"`
}
