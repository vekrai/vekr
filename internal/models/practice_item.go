package models

import (
	"regexp"
	"strings"
	"time"
)

const (
	PracticeItemTypeMCQ       = "mcq"
	PracticeItemTypeFlashcard = "flashcard"
	PracticeItemTypeOpen      = "open"
	PracticeItemTypeCode      = "code"
)

type PracticeItem struct {
	ID        uint `gorm:"primaryKey"`
	TopicID   uint
	Prompt    string
	AnswerKey *string
	Type      string
	CreatedAt time.Time
}

type PracticeItemConcept struct {
	PracticeItemID uint `gorm:"primaryKey"`
	ConceptID      uint `gorm:"primaryKey"`
}

type Choice struct {
	Letter string
	Text   string
}

var choicePattern = regexp.MustCompile(`^\s*\(?([A-Ha-h])[\)\.:]\s+(.+?)\s*$`)

func (p PracticeItem) Choices() (string, []Choice) {
	if p.Type != PracticeItemTypeMCQ {
		return p.Prompt, nil
	}
	var stem []string
	var choices []Choice
	for _, line := range strings.Split(p.Prompt, "\n") {
		if m := choicePattern.FindStringSubmatch(line); m != nil {
			choices = append(choices, Choice{Letter: strings.ToUpper(m[1]), Text: m[2]})
			continue
		}
		stem = append(stem, line)
	}
	if len(choices) < 2 {
		return p.Prompt, nil
	}
	return strings.TrimSpace(strings.Join(stem, "\n")), choices
}

func (p PracticeItem) ExpectedAnswer() string {
	if p.AnswerKey == nil {
		return ""
	}
	key := strings.TrimSpace(*p.AnswerKey)
	_, choices := p.Choices()
	for _, c := range choices {
		if strings.EqualFold(c.Letter, key) {
			return c.Letter + ") " + c.Text
		}
	}
	return key
}
