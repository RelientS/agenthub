package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

const (
	anthropicAPIURL   = "https://api.anthropic.com/v1/messages"
	anthropicModel    = "claude-sonnet-4-5"
	anthropicVersion  = "2023-06-01"
	maxSubTasks       = 10
	llmRequestTimeout = 60 * time.Second
)

// SubTaskSuggestion represents a suggested subtask returned by the LLM.
type SubTaskSuggestion struct {
	Title          string  `json:"title"`
	Description    string  `json:"description"`
	EstimatedHours float64 `json:"estimated_hours"`
	DependsOnIndex []int   `json:"depends_on_index,omitempty"` // 0-based indices into the suggestions slice
}

// LLMService handles AI-powered features via the Anthropic API.
type LLMService struct {
	apiKey     string
	httpClient *http.Client
}

// NewLLMService creates a new LLMService, reading ANTHROPIC_API_KEY from the environment.
func NewLLMService() *LLMService {
	return &LLMService{
		apiKey: os.Getenv("ANTHROPIC_API_KEY"),
		httpClient: &http.Client{
			Timeout: llmRequestTimeout,
		},
	}
}

// DecomposeTask calls the Anthropic API and returns subtask suggestions for the given task.
func (s *LLMService) DecomposeTask(ctx context.Context, title, description string) ([]SubTaskSuggestion, error) {
	if s.apiKey == "" {
		return nil, fmt.Errorf("ANTHROPIC_API_KEY is not configured")
	}

	prompt := buildDecomposePrompt(title, description)

	reqBody := map[string]interface{}{
		"model":      anthropicModel,
		"max_tokens": 2048,
		"messages": []map[string]string{
			{"role": "user", "content": prompt},
		},
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshaling LLM request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, anthropicAPIURL, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("building LLM request: %w", err)
	}
	req.Header.Set("content-type", "application/json")
	req.Header.Set("x-api-key", s.apiKey)
	req.Header.Set("anthropic-version", anthropicVersion)

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("calling Anthropic API: %w", err)
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading Anthropic API response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Anthropic API returned status %d: %s", resp.StatusCode, string(respBytes))
	}

	var apiResp struct {
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
	}
	if err := json.Unmarshal(respBytes, &apiResp); err != nil {
		return nil, fmt.Errorf("parsing Anthropic API response: %w", err)
	}

	if len(apiResp.Content) == 0 {
		return nil, fmt.Errorf("empty content in Anthropic API response")
	}

	suggestions, err := parseSubTaskJSON(apiResp.Content[0].Text)
	if err != nil {
		return nil, fmt.Errorf("parsing subtask JSON from LLM: %w", err)
	}

	if len(suggestions) > maxSubTasks {
		suggestions = suggestions[:maxSubTasks]
	}

	return suggestions, nil
}

// buildDecomposePrompt constructs the user prompt for task decomposition.
func buildDecomposePrompt(title, description string) string {
	var sb strings.Builder
	sb.WriteString("You are a software project manager. Break down the following task into actionable subtasks.\n\n")
	sb.WriteString("Task Title: ")
	sb.WriteString(title)
	sb.WriteString("\n")
	if description != "" {
		sb.WriteString("Task Description: ")
		sb.WriteString(description)
		sb.WriteString("\n")
	}
	sb.WriteString(`
Return ONLY a JSON array (no markdown code fences, no explanation text) containing at most 10 subtasks.
Each element must have exactly these fields:
- "title": string (concise, imperative verb phrase)
- "description": string (what needs to be done and acceptance criteria)
- "estimated_hours": number (realistic work estimate in hours)
- "depends_on_index": array of integers (0-based indices of subtasks this one depends on; use [] if none)

Example:
[
  {"title": "Design database schema", "description": "Define tables, columns, and indexes", "estimated_hours": 2, "depends_on_index": []},
  {"title": "Implement repository layer", "description": "CRUD operations for each entity", "estimated_hours": 4, "depends_on_index": [0]}
]`)
	return sb.String()
}

// parseSubTaskJSON extracts a JSON array of SubTaskSuggestion from LLM response text.
// It tolerates extra text before or after the JSON array.
func parseSubTaskJSON(text string) ([]SubTaskSuggestion, error) {
	start := strings.Index(text, "[")
	end := strings.LastIndex(text, "]")
	if start == -1 || end == -1 || end <= start {
		return nil, fmt.Errorf("no JSON array found in LLM response")
	}

	var suggestions []SubTaskSuggestion
	if err := json.Unmarshal([]byte(text[start:end+1]), &suggestions); err != nil {
		return nil, fmt.Errorf("invalid JSON array in LLM response: %w", err)
	}

	return suggestions, nil
}
