package agent

import (
	"context"
	_ "embed"
	"fmt"
	"strings"

	"gorm.io/gorm"

	"vekr/internal/ai"
	"vekr/internal/models"
)

//go:embed prompts/curriculum_system.txt
var curriculumSystemPromptFile string

var curriculumSystemPrompt = strings.TrimSpace(curriculumSystemPromptFile)

type CurriculumService struct {
	db     *gorm.DB
	client *ai.Client
}

func NewCurriculumService(db *gorm.DB, client *ai.Client) *CurriculumService {
	return &CurriculumService{db: db, client: client}
}

type CurriculumPreview struct {
	PathTitle string         `json:"path_title"`
	Topics    []PreviewTopic `json:"topics"`
}

type PreviewTopic struct {
	Title         string   `json:"title"`
	Content       string   `json:"content"`
	Prerequisites []string `json:"prerequisites"`
}

// Preview turns a goal or pasted material into a proposed curriculum — a
// Path plus a batch of prerequisite-linked Topics — without persisting
// anything. The caller holds the result (e.g. in the session) until the
// user approves it via Commit.
func (s *CurriculumService) Preview(ctx context.Context, intake string) (CurriculumPreview, error) {
	req := []ai.Message{
		{Role: ai.RoleSystem, Content: curriculumSystemPrompt},
		{Role: ai.RoleUser, Content: intake},
	}

	preview, err := CallStructured[CurriculumPreview](ctx, s.client, req)
	if err != nil {
		return CurriculumPreview{}, fmt.Errorf("agent: curriculum preview: %w", err)
	}
	if preview.PathTitle == "" || len(preview.Topics) == 0 {
		return CurriculumPreview{}, fmt.Errorf("agent: curriculum preview: model returned no path title or no topics")
	}
	return preview, nil
}

// Commit persists a previously generated CurriculumPreview: one Path row,
// one Topic node per preview topic, and a prerequisite_of node_link per
// declared prerequisite. Runs in a transaction so a partial graph is never
// left behind on failure.
func (s *CurriculumService) Commit(ctx context.Context, preview CurriculumPreview) (models.Path, error) {
	var path models.Path

	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		path = models.Path{Title: preview.PathTitle}
		if err := tx.Create(&path).Error; err != nil {
			return fmt.Errorf("save path: %w", err)
		}

		topicsByTitle := make(map[string]models.Node, len(preview.Topics))
		for _, topic := range preview.Topics {
			pathID := path.ID
			status := models.StatusNotStarted
			node := models.Node{
				Title:   topic.Title,
				Content: topic.Content,
				Type:    models.NodeTypeTopic,
				PathID:  &pathID,
				Status:  &status,
			}
			if err := tx.Create(&node).Error; err != nil {
				return fmt.Errorf("save topic %q: %w", topic.Title, err)
			}
			topicsByTitle[topic.Title] = node
		}

		for _, topic := range preview.Topics {
			node := topicsByTitle[topic.Title]
			for _, prereqTitle := range topic.Prerequisites {
				prereq, ok := topicsByTitle[prereqTitle]
				if !ok {
					// The model referenced a prerequisite outside this
					// batch — skip the link rather than fail the whole
					// commit over one bad cross-reference.
					continue
				}
				link := models.NodeLink{FromNodeID: prereq.ID, ToNodeID: node.ID, Relation: models.RelationPrerequisiteOf}
				if err := tx.Create(&link).Error; err != nil {
					return fmt.Errorf("save prerequisite link %q -> %q: %w", prereqTitle, topic.Title, err)
				}
			}
		}

		return nil
	})
	if err != nil {
		return models.Path{}, fmt.Errorf("agent: curriculum commit: %w", err)
	}

	return path, nil
}

// PromoteToTopic turns an existing Note into a Topic — a metadata update
// only, per the Note/Topic design (same row, no migration). It does not
// assign a Path; that stays nil until the note is added to one.
func (s *CurriculumService) PromoteToTopic(ctx context.Context, nodeID uint) (models.Node, error) {
	updates := map[string]any{
		"type":   models.NodeTypeTopic,
		"status": models.StatusNotStarted,
	}
	if err := s.db.WithContext(ctx).Model(&models.Node{}).Where("id = ?", nodeID).Updates(updates).Error; err != nil {
		return models.Node{}, fmt.Errorf("agent: promote to topic: %w", err)
	}

	var node models.Node
	if err := s.db.WithContext(ctx).First(&node, nodeID).Error; err != nil {
		return models.Node{}, fmt.Errorf("agent: promote to topic: reload: %w", err)
	}
	return node, nil
}
