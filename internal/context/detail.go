package context

import (
	"fmt"
	"knodexa/internal/model"
	"strings"
)

// firstParagraph preserves a complete paragraph rather than cutting off an
// instruction mid-clause. Authors can supply an explicit concise summary.
func firstParagraph(body string) string {
	lines := strings.Split(strings.TrimSpace(body), "\n")
	var paragraph []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "#") || line == "" {
			if len(paragraph) > 0 {
				break
			}
			continue
		}
		paragraph = append(paragraph, line)
	}
	return strings.Join(paragraph, " ")
}

func decisionConstraints(doc *model.DecisionDocument) []string {
	result := append([]string{}, doc.Frontmatter.Constraints...)
	// Retain constraints written by older handlers before structured frontmatter.
	inSection := false
	for _, line := range strings.Split(doc.Body, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "#") {
			inSection = strings.EqualFold(strings.TrimSpace(strings.TrimLeft(line, "#")), "Constraints")
			continue
		}
		if inSection && line != "" {
			result = append(result, strings.TrimSpace(strings.TrimPrefix(line, "- ")))
		}
	}
	return result
}

// GetDetail expands a decision or lesson without loading unrelated knowledge.
func (b *Builder) GetDetail(id string) (interface{}, error) {
	if strings.HasPrefix(id, "D-") {
		return b.db.GetDecisionDocument(id)
	}
	if strings.HasPrefix(id, "L-") {
		return b.db.GetLessonDocument(id)
	}
	return nil, fmt.Errorf("detail supports decision (D-) and lesson (L-) IDs")
}
