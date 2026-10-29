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

func NewToolRegistry(webSearch *WebSearchTool, webFetch *WebFetchTool, knowledge *KnowledgeTools) *ToolRegistry {
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
