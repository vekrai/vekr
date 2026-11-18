package agent

import (
	"context"
	"encoding/json"
	"fmt"

	"vekr/internal/ai"
)

const maxToolSteps = 8

type ToolHandler func(ctx context.Context, args json.RawMessage) (string, error)

type ToolRegistry struct {
	definitions []ai.Tool
	handlers    map[string]ToolHandler
}

func NewToolRegistry(webSearch *WebSearchTool, webFetch *WebFetchTool, knowledge *KnowledgeTools, curriculum *CurriculumTools, practice *PracticeTools) *ToolRegistry {
	r := &ToolRegistry{handlers: make(map[string]ToolHandler)}

	r.register(ai.Tool{
		Type: "function",
		Function: ai.ToolFunction{
			Name:        "websearch",
			Description: "Search the web for current information. Returns a ranked list of results with titles, URLs, and snippets.",
			Parameters: json.RawMessage(`{
				"type": "object",
				"properties": { "query": { "type": "string", "description": "The search query" } },
				"required": ["query"]
			}`),
		},
	}, func(ctx context.Context, args json.RawMessage) (string, error) {
		var parsed struct {
			Query string `json:"query"`
		}
		if err := json.Unmarshal(args, &parsed); err != nil {
			return "", fmt.Errorf("invalid arguments: %w", err)
		}
		if parsed.Query == "" {
			return "", fmt.Errorf(`invalid arguments: "query" is required`)
		}
		return webSearch.Search(ctx, parsed.Query)
	})

	r.register(ai.Tool{
		Type: "function",
		Function: ai.ToolFunction{
			Name:        "webfetch",
			Description: "Fetch a URL and return its content as plain text.",
			Parameters: json.RawMessage(`{
				"type": "object",
				"properties": { "url": { "type": "string", "description": "The http(s) URL to fetch" } },
				"required": ["url"]
			}`),
		},
	}, func(ctx context.Context, args json.RawMessage) (string, error) {
		var parsed struct {
			URL string `json:"url"`
		}
		if err := json.Unmarshal(args, &parsed); err != nil {
			return "", fmt.Errorf("invalid arguments: %w", err)
		}
		if parsed.URL == "" {
			return "", fmt.Errorf(`invalid arguments: "url" is required`)
		}
		return webFetch.Fetch(ctx, parsed.URL)
	})

	r.register(ai.Tool{
		Type: "function",
		Function: ai.ToolFunction{
			Name:        "search_notes",
			Description: "Search the user's own notes by keyword. Returns a ranked list of node_id, title, and a matched snippet per note.",
			Parameters: json.RawMessage(`{
				"type": "object",
				"properties": { "query": { "type": "string", "description": "Keywords to search for" } },
				"required": ["query"]
			}`),
		},
	}, func(ctx context.Context, args json.RawMessage) (string, error) {
		var parsed struct {
			Query string `json:"query"`
		}
		if err := json.Unmarshal(args, &parsed); err != nil {
			return "", fmt.Errorf("invalid arguments: %w", err)
		}
		if parsed.Query == "" {
			return "", fmt.Errorf(`invalid arguments: "query" is required`)
		}
		return knowledge.SearchNotes(ctx, parsed.Query)
	})

	r.register(ai.Tool{
		Type: "function",
		Function: ai.ToolFunction{
			Name:        "read_note",
			Description: "Read the full content of one note, given a node_id from a prior search_notes result.",
			Parameters: json.RawMessage(`{
				"type": "object",
				"properties": { "node_id": { "type": "integer", "description": "The node_id to read" } },
				"required": ["node_id"]
			}`),
		},
	}, func(ctx context.Context, args json.RawMessage) (string, error) {
		var parsed struct {
			NodeID uint `json:"node_id"`
		}
		if err := json.Unmarshal(args, &parsed); err != nil {
			return "", fmt.Errorf("invalid arguments: %w", err)
		}
		if parsed.NodeID == 0 {
			return "", fmt.Errorf(`invalid arguments: "node_id" is required`)
		}
		return knowledge.ReadNote(ctx, parsed.NodeID)
	})

	r.register(ai.Tool{
		Type: "function",
		Function: ai.ToolFunction{
			Name:        "edit_note",
			Description: "Edit an existing note's title and/or content, given a node_id from a prior search_notes or read_note call. Only the fields you pass are changed — omit a field to leave it as-is. Commits directly, no confirmation step.",
			Parameters: json.RawMessage(`{
				"type": "object",
				"properties": {
					"node_id": { "type": "integer", "description": "The node_id of the note to edit" },
					"title": { "type": "string", "description": "New title, if changing it" },
					"content": { "type": "string", "description": "New content, if changing it" }
				},
				"required": ["node_id"]
			}`),
		},
	}, func(ctx context.Context, args json.RawMessage) (string, error) {
		var parsed struct {
			NodeID  uint    `json:"node_id"`
			Title   *string `json:"title"`
			Content *string `json:"content"`
		}
		if err := json.Unmarshal(args, &parsed); err != nil {
			return "", fmt.Errorf("invalid arguments: %w", err)
		}
		if parsed.NodeID == 0 {
			return "", fmt.Errorf(`invalid arguments: "node_id" is required`)
		}
		return knowledge.EditNote(ctx, parsed.NodeID, parsed.Title, parsed.Content)
	})

	r.register(ai.Tool{
		Type: "function",
		Function: ai.ToolFunction{
			Name:        "generate_curriculum_preview",
			Description: "Generate a proposed topic (a title plus a batch of outline notes with prerequisites) from a goal or pasted material. Nothing is saved yet — show the result to the user and ask them to confirm before calling commit_curriculum_preview.",
			Parameters: json.RawMessage(`{
				"type": "object",
				"properties": { "intake": { "type": "string", "description": "The learning goal, or material to build a curriculum from" } },
				"required": ["intake"]
			}`),
		},
	}, func(ctx context.Context, args json.RawMessage) (string, error) {
		var parsed struct {
			Intake string `json:"intake"`
		}
		if err := json.Unmarshal(args, &parsed); err != nil {
			return "", fmt.Errorf("invalid arguments: %w", err)
		}
		if parsed.Intake == "" {
			return "", fmt.Errorf(`invalid arguments: "intake" is required`)
		}
		return curriculum.GeneratePreview(ctx, parsed.Intake)
	})

	r.register(ai.Tool{
		Type: "function",
		Function: ai.ToolFunction{
			Name:        "commit_curriculum_preview",
			Description: "Save the most recently generated curriculum preview as a real topic with its notes and prerequisite links. Only call this after the user has explicitly confirmed the preview looks right.",
			Parameters:  json.RawMessage(`{"type": "object", "properties": {}}`),
		},
	}, func(ctx context.Context, args json.RawMessage) (string, error) {
		return curriculum.CommitPreview(ctx)
	})

	r.register(ai.Tool{
		Type: "function",
		Function: ai.ToolFunction{
			Name:        "generate_practice_items",
			Description: "Generate practice questions for a topic, given its topic_id. Items are saved immediately — cheap to regenerate, no confirmation needed.",
			Parameters: json.RawMessage(`{
				"type": "object",
				"properties": {
					"topic_id": { "type": "integer", "description": "The topic's topic_id" },
					"count": { "type": "integer", "description": "How many items to generate (default 5 if omitted)" }
				},
				"required": ["topic_id"]
			}`),
		},
	}, func(ctx context.Context, args json.RawMessage) (string, error) {
		var parsed struct {
			TopicID uint `json:"topic_id"`
			Count   int  `json:"count"`
		}
		if err := json.Unmarshal(args, &parsed); err != nil {
			return "", fmt.Errorf("invalid arguments: %w", err)
		}
		if parsed.TopicID == 0 {
			return "", fmt.Errorf(`invalid arguments: "topic_id" is required`)
		}
		count := parsed.Count
		if count <= 0 {
			count = 5
		}
		return practice.GenerateItems(ctx, parsed.TopicID, count)
	})

	return r
}

func (r *ToolRegistry) register(def ai.Tool, handler ToolHandler) {
	r.definitions = append(r.definitions, def)
	r.handlers[def.Function.Name] = handler
}

func (r *ToolRegistry) Definitions() []ai.Tool {
	return r.definitions
}

func (r *ToolRegistry) Dispatch(ctx context.Context, name string, args json.RawMessage) (string, error) {
	handler, ok := r.handlers[name]
	if !ok {
		return "", fmt.Errorf("unknown tool %q", name)
	}
	return handler(ctx, args)
}
