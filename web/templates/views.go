package templates

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"vekr/internal/ai"
	"vekr/internal/markdown"
	"vekr/internal/models"
)

type TopicView struct {
	Progress    models.TopicProgress
	ContentHTML string
	Notes       []models.Node
	Concepts    []models.ConceptStat
	Attempts    []models.AttemptResult
}

type NoteView struct {
	Node        models.Node
	Topic       *models.Topic
	ContentHTML string
	Keywords    []string
	Related     []models.RelatedNote
	Chats       []models.ChatSummary
}

type NotesIndexView struct {
	Query       string
	Nodes       []models.Node
	Hits        []models.NoteHit
	TopicTitles map[uint]string
}

type PracticeView struct {
	Topic       *models.Topic
	Concepts    []models.Concept
	ConceptIDs  []uint
	Items       []models.PracticeItem
	Latest      map[uint]models.AttemptResult
	TopicTitles map[uint]string
	ReturnTo    string
	Graded      uint
}

type crumb struct {
	Label string
	URL   string
}

func dashboardCrumb(ctx context.Context) crumb {
	return crumb{Label: t(ctx, "nav.dashboard"), URL: "/"}
}

func topicCrumb(topic *models.Topic) crumb {
	return crumb{Label: topic.Title, URL: fmt.Sprintf("/topics/%d", topic.ID)}
}

func relativeDate(ctx context.Context, at time.Time) string {
	now := time.Now()
	y1, m1, d1 := now.Date()
	today := time.Date(y1, m1, d1, 0, 0, 0, 0, now.Location())
	local := at.In(now.Location())
	switch {
	case !local.Before(today):
		return t(ctx, "date.today")
	case !local.Before(today.AddDate(0, 0, -1)):
		return t(ctx, "date.yesterday")
	case local.Year() == now.Year():
		return local.Format("Jan 2")
	default:
		return local.Format("Jan 2, 2006")
	}
}

func humanize(name string) string {
	return models.HumanizeConcept(name)
}

func markdownHTML(source string) string {
	html, err := markdown.ToHTML(source)
	if err != nil {
		return ""
	}
	return html
}

func firstLine(text string) string {
	for _, line := range strings.Split(text, "\n") {
		if line = strings.TrimSpace(line); line != "" {
			return line
		}
	}
	return ""
}

func plainSnippet(snippet string) string {
	return strings.NewReplacer("<b>", "", "</b>", "", "**", "", "*", "", "`", "", "# ", "").Replace(snippet)
}

func percent(passed, total int64) float64 {
	if total == 0 {
		return 0
	}
	return float64(passed) / float64(total) * 100
}

func itemTypeLabel(ctx context.Context, itemType string) string {
	return t(ctx, "practice.type."+itemType)
}

func verdictLabel(ctx context.Context, a models.AttemptResult) string {
	return t(ctx, "practice.verdict."+a.Verdict())
}

func verdictClass(a models.AttemptResult) string {
	switch a.Verdict() {
	case models.VerdictPassed:
		return "chip chip-good"
	case models.VerdictPassedWithNote:
		return "chip chip-warning"
	default:
		return "chip chip-serious"
	}
}

func relationLabel(ctx context.Context, r models.RelatedNote) string {
	direction := "in"
	if r.Outgoing {
		direction = "out"
	}
	return t(ctx, "note.relation."+r.Relation+"."+direction)
}

func compactTokens(n int) string {
	if n < 1000 {
		return fmt.Sprintf("%d", n)
	}
	return strings.TrimSuffix(fmt.Sprintf("%.1f", float64(n)/1000), ".0") + "k"
}

func tokenUsage(ctx context.Context, state ChatState) string {
	if state.TokensUsed == 0 {
		return ""
	}
	if state.ContextWindow == 0 {
		return tData(ctx, "chat.tokens", map[string]any{"Used": compactTokens(state.TokensUsed)})
	}
	return tData(ctx, "chat.tokens_of", map[string]any{
		"Used":    compactTokens(state.TokensUsed),
		"Window":  compactTokens(state.ContextWindow),
		"Percent": state.TokensUsed * 100 / state.ContextWindow,
	})
}

var knownTools = map[string]bool{
	"websearch":                   true,
	"webfetch":                    true,
	"search_notes":                true,
	"read_note":                   true,
	"edit_note":                   true,
	"generate_curriculum_preview": true,
	"commit_curriculum_preview":   true,
	"generate_practice_items":     true,
}

type toolActivity struct {
	Label  string
	Detail string
	Result string
}

func toolResults(ctx context.Context, messages []models.ChatMessage) map[string]string {
	results := map[string]string{}
	for _, m := range messages {
		if m.Role == "tool" {
			results[m.ToolCallID] = toolResultSummary(ctx, m)
		}
	}
	return results
}

func toolCalls(ctx context.Context, m models.ChatMessage, results map[string]string) []toolActivity {
	if m.ToolCalls == "" {
		return nil
	}
	var calls []ai.ToolCall
	if err := json.Unmarshal([]byte(m.ToolCalls), &calls); err != nil {
		return nil
	}
	activities := make([]toolActivity, 0, len(calls))
	for _, c := range calls {
		var args map[string]any
		_ = json.Unmarshal([]byte(c.Function.Arguments), &args)
		detail := ""
		for _, key := range []string{"query", "url", "title"} {
			if v, ok := args[key]; ok {
				detail = fmt.Sprint(v)
				break
			}
		}
		label := tData(ctx, "chat.tool.generic", map[string]any{"Name": c.Function.Name})
		if knownTools[c.Function.Name] {
			label = t(ctx, "chat.tool."+c.Function.Name)
		}
		activities = append(activities, toolActivity{Label: label, Detail: detail, Result: results[c.ID]})
	}
	return activities
}

func toolResultSummary(ctx context.Context, m models.ChatMessage) string {
	if strings.HasPrefix(m.Content, "error:") {
		return t(ctx, "chat.tool.failed")
	}
	if m.Name == "read_note" && strings.HasPrefix(m.Content, "Title: ") {
		return firstLine(strings.TrimPrefix(m.Content, "Title: "))
	}
	if m.Name == "search_notes" {
		if m.Content == "no matching notes found" {
			return tCount(ctx, "chat.tool.results", 0, nil)
		}
		return tCount(ctx, "chat.tool.results", strings.Count(m.Content, "node_id="), nil)
	}
	return ""
}
