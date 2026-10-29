package ai

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sort"
	"strings"
)

type StreamEvent struct {
	ContentDelta string
}

type streamChunk struct {
	Choices []struct {
		Delta        streamDelta `json:"delta"`
		FinishReason *string     `json:"finish_reason"`
	} `json:"choices"`
	Usage *Usage `json:"usage"`
}

type streamDelta struct {
	Role      Role            `json:"role,omitempty"`
	Content   string          `json:"content,omitempty"`
	ToolCalls []toolCallDelta `json:"tool_calls,omitempty"`
}

type toolCallDelta struct {
	Index    int    `json:"index"`
	ID       string `json:"id,omitempty"`
	Type     string `json:"type,omitempty"`
	Function struct {
		Name      string `json:"name,omitempty"`
		Arguments string `json:"arguments,omitempty"`
	} `json:"function,omitempty"`
}

func (c *Client) StreamChatCompletion(ctx context.Context, req ChatRequest, onEvent func(StreamEvent)) (*ChatResponse, error) {
	httpReq, err := c.newRequest(ctx, c.payload(req, true))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Accept", "text/event-stream")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("ai: request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("ai: unexpected status %d: %s", resp.StatusCode, readErrorBody(resp))
	}

	acc := newStreamAccumulator()

	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		line := scanner.Text()
		data, ok := strings.CutPrefix(line, "data:")
		if !ok {
			continue
		}
		data = strings.TrimSpace(data)
		if data == "" {
			continue
		}
		if data == "[DONE]" {
			break
		}

		var chunk streamChunk
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			return nil, fmt.Errorf("ai: decode stream chunk: %w", err)
		}
		acc.apply(chunk)

		if onEvent != nil {
			for _, choice := range chunk.Choices {
				if choice.Delta.Content != "" {
					onEvent(StreamEvent{ContentDelta: choice.Delta.Content})
				}
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("ai: read stream: %w", err)
	}

	result := acc.result()
	log.Printf("ai: stream finish_reason=%s usage=%+v content=%s", result.FinishReason, result.Usage, truncateForLog([]byte(result.Message.Content)))
	return result, nil
}

type streamAccumulator struct {
	content      strings.Builder
	toolCalls    map[int]*accumulatingToolCall
	finishReason string
	usage        Usage
}

type accumulatingToolCall struct {
	id        string
	toolType  string
	name      string
	arguments strings.Builder
}

func newStreamAccumulator() *streamAccumulator {
	return &streamAccumulator{toolCalls: make(map[int]*accumulatingToolCall)}
}

func (a *streamAccumulator) apply(chunk streamChunk) {
	if chunk.Usage != nil {
		a.usage = *chunk.Usage
	}

	for _, choice := range chunk.Choices {
		a.content.WriteString(choice.Delta.Content)

		for _, tc := range choice.Delta.ToolCalls {
			call, ok := a.toolCalls[tc.Index]
			if !ok {
				call = &accumulatingToolCall{}
				a.toolCalls[tc.Index] = call
			}
			if tc.ID != "" {
				call.id = tc.ID
			}
			if tc.Type != "" {
				call.toolType = tc.Type
			}
			if tc.Function.Name != "" {
				call.name = tc.Function.Name
			}
			call.arguments.WriteString(tc.Function.Arguments)
		}

		if choice.FinishReason != nil {
			a.finishReason = *choice.FinishReason
		}
	}
}

func (a *streamAccumulator) result() *ChatResponse {
	indices := make([]int, 0, len(a.toolCalls))
	for i := range a.toolCalls {
		indices = append(indices, i)
	}
	sort.Ints(indices)

	toolCalls := make([]ToolCall, 0, len(indices))
	for _, i := range indices {
		call := a.toolCalls[i]
		toolCalls = append(toolCalls, ToolCall{
			ID:   call.id,
			Type: call.toolType,
			Function: ToolCallFunction{
				Name:      call.name,
				Arguments: call.arguments.String(),
			},
		})
	}

	return &ChatResponse{
		Message: Message{
			Role:      RoleAssistant,
			Content:   a.content.String(),
			ToolCalls: toolCalls,
		},
		FinishReason: a.finishReason,
		Usage:        a.usage,
	}
}
