package ai

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

const (
	anthropicEndpoint = "https://api.anthropic.com/v1/messages"
	anthropicVersion  = "2023-06-01"
	defaultOllamaURL  = "http://127.0.0.1:11434"

	// DefaultModel is the model used for Anthropic calls.
	DefaultModel = "claude-sonnet-4-20250514"
	// DefaultOllamaModel is the default free local model for Ollama.
	DefaultOllamaModel = "qwen2.5-coder:1.5b"

	defaultMaxTokens = 2048
	// streamBufSize is the channel buffer for streaming tokens.
	// Large enough to absorb burst writes without blocking the HTTP goroutine.
	streamBufSize = 64
)

type completionClient interface {
	complete(ctx context.Context, system, prompt string, maxTokens int) (string, error)
	stream(ctx context.Context, system, prompt string, maxTokens int) StreamResult
}

type apiMessage struct {
	Role    string `json:"role"` // "user" or "assistant"
	Content string `json:"content"`
}

type anthropicRequest struct {
	Model string `json:"model"`
	//nolint:tagliatelle // Anthropic API requires snake_case.
	MaxTokens int          `json:"max_tokens"`
	System    string       `json:"system,omitempty"`
	Messages  []apiMessage `json:"messages"`
	Stream    bool         `json:"stream,omitempty"`
}

type anthropicResponse struct {
	Content []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"content"`
	Error *struct {
		Type    string `json:"type"`
		Message string `json:"message"`
	} `json:"error"`
}

type ollamaRequest struct {
	Model    string       `json:"model"`
	Messages []apiMessage `json:"messages"`
	Stream   bool         `json:"stream"`
}

type ollamaResponse struct {
	Message struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	} `json:"message"`
	Done bool `json:"done"`
	//nolint:tagliatelle // Ollama API uses snake_case.
	DoneReason string `json:"done_reason"`
	Error      string `json:"error"`
}

type anthropicClient struct {
	apiKey     string
	model      string
	httpClient *http.Client
}

type ollamaClient struct {
	baseURL    string
	model      string
	httpClient *http.Client
}

func newAnthropicClient(apiKey, model string) *anthropicClient {
	if strings.TrimSpace(apiKey) == "" {
		return nil
	}
	if model == "" {
		model = DefaultModel
	}
	return &anthropicClient{
		apiKey: apiKey,
		model:  model,
		httpClient: &http.Client{
			Timeout: 90 * time.Second,
		},
	}
}

func newOllamaClient(baseURL, model string) *ollamaClient {
	baseURL = normalizeOllamaBaseURL(baseURL)
	model = strings.TrimSpace(model)
	if model == "" {
		model = strings.TrimSpace(os.Getenv("GITFLOW_TUI_OLLAMA_MODEL"))
	}
	if model == "" {
		model = DefaultOllamaModel
	}
	return &ollamaClient{
		baseURL: baseURL,
		model:   model,
		httpClient: &http.Client{
			Timeout: 60 * time.Second,
		},
	}
}

func normalizeOllamaBaseURL(baseURL string) string {
	baseURL = strings.TrimSpace(baseURL)
	if baseURL == "" {
		baseURL = strings.TrimSpace(os.Getenv("GITFLOW_TUI_OLLAMA_BASE_URL"))
	}
	if baseURL == "" {
		baseURL = defaultOllamaURL
	}
	return strings.TrimRight(baseURL, "/")
}

func ollamaReachable(ctx context.Context, baseURL string) bool {
	client := &http.Client{Timeout: 500 * time.Millisecond}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, normalizeOllamaBaseURL(baseURL)+"/api/tags", nil)
	if err != nil {
		return false
	}

	resp, err := client.Do(req)
	if err != nil {
		return false
	}
	if closeErr := resp.Body.Close(); closeErr != nil {
		return false
	}

	return resp.StatusCode >= 200 && resp.StatusCode < 500
}

// complete sends a single-turn prompt and returns the full response text.
// It is used for structured-JSON responses where we need the complete output
// before parsing (commit suggestions, conflict analysis, health reports).
func (c *anthropicClient) complete(ctx context.Context, system, prompt string, maxTokens int) (string, error) {
	if maxTokens <= 0 {
		maxTokens = defaultMaxTokens
	}

	payload, err := json.Marshal(anthropicRequest{
		Model:     c.model,
		MaxTokens: maxTokens,
		System:    system,
		Messages:  []apiMessage{{Role: "user", Content: prompt}},
	})
	if err != nil {
		return "", fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, anthropicEndpoint, bytes.NewReader(payload))
	if err != nil {
		return "", fmt.Errorf("build request: %w", err)
	}
	c.setHeaders(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("send request: %w", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	var result anthropicResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("decode response (status %d): %w", resp.StatusCode, err)
	}
	if result.Error != nil {
		return "", &APIError{Backend: "anthropic", Type: result.Error.Type, Message: result.Error.Message}
	}
	if len(result.Content) == 0 {
		return "", errors.New("empty content in API response")
	}
	return result.Content[0].Text, nil
}

// StreamResult contains streaming tokens and a final error channel.
type StreamResult struct {
	// Tokens receives text tokens as they arrive from the API.
	// The channel is closed when the stream ends (success or error).
	Tokens <-chan string
	// Err receives at most one error, then is closed.
	// If nil is received the stream completed successfully.
	Err <-chan error
}

func (c *anthropicClient) stream(ctx context.Context, system, prompt string, maxTokens int) StreamResult {
	tokenCh := make(chan string, streamBufSize)
	errCh := make(chan error, 1)

	if maxTokens <= 0 {
		maxTokens = defaultMaxTokens
	}

	go func() {
		defer close(tokenCh)
		defer close(errCh)

		payload, err := json.Marshal(anthropicRequest{
			Model:     c.model,
			MaxTokens: maxTokens,
			System:    system,
			Messages:  []apiMessage{{Role: "user", Content: prompt}},
			Stream:    true,
		})
		if err != nil {
			errCh <- fmt.Errorf("marshal stream request: %w", err)
			return
		}

		req, err := http.NewRequestWithContext(ctx, http.MethodPost, anthropicEndpoint, bytes.NewReader(payload))
		if err != nil {
			errCh <- fmt.Errorf("build stream request: %w", err)
			return
		}
		c.setHeaders(req)

		resp, err := (&http.Client{}).Do(req)
		if err != nil {
			errCh <- fmt.Errorf("send stream request: %w", err)
			return
		}
		defer func() {
			_ = resp.Body.Close()
		}()

		scanner := bufio.NewScanner(resp.Body)
		for scanner.Scan() {
			line := scanner.Text()
			if !strings.HasPrefix(line, "data: ") {
				continue
			}
			data := strings.TrimPrefix(line, "data: ")
			if data == "[DONE]" {
				return
			}

			var event struct {
				Type  string `json:"type"`
				Delta struct {
					Type string `json:"type"`
					Text string `json:"text"`
				} `json:"delta"`
				Error *struct {
					Type    string `json:"type"`
					Message string `json:"message"`
				} `json:"error"`
			}
			if err := json.Unmarshal([]byte(data), &event); err != nil {
				continue
			}

			if event.Error != nil {
				errCh <- &APIError{Backend: "anthropic", Type: event.Error.Type, Message: event.Error.Message}
				return
			}

			if event.Type == "content_block_delta" && event.Delta.Type == "text_delta" {
				select {
				case tokenCh <- event.Delta.Text:
				case <-ctx.Done():
					return
				}
			}
		}

		if err := scanner.Err(); err != nil {
			errCh <- fmt.Errorf("read stream response: %w", err)
		}
	}()

	return StreamResult{Tokens: tokenCh, Err: errCh}
}

func (c *anthropicClient) setHeaders(req *http.Request) {
	req.Header.Set("X-Api-Key", c.apiKey)
	req.Header.Set("Anthropic-Version", anthropicVersion)
	req.Header.Set("Content-Type", "application/json")
}

func (c *ollamaClient) complete(ctx context.Context, system, prompt string, maxTokens int) (string, error) {
	req, err := c.newRequest(ctx, system, prompt, false)
	if err != nil {
		return "", err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("send request: %w", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, readErr := io.ReadAll(resp.Body)
		if readErr != nil {
			return "", fmt.Errorf("ollama status %d: read error body: %w", resp.StatusCode, readErr)
		}
		return "", &APIError{
			Backend: "ollama",
			Type:    fmt.Sprintf("status_%d", resp.StatusCode),
			Message: strings.TrimSpace(string(body)),
		}
	}

	var result ollamaResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("decode response: %w", err)
	}
	if strings.TrimSpace(result.Error) != "" {
		return "", &APIError{Backend: "ollama", Type: "request_failed", Message: result.Error}
	}

	return result.Message.Content, nil
}

func (c *ollamaClient) stream(ctx context.Context, system, prompt string, maxTokens int) StreamResult {
	tokenCh := make(chan string, streamBufSize)
	errCh := make(chan error, 1)

	go func() {
		defer close(tokenCh)
		defer close(errCh)

		req, err := c.newRequest(ctx, system, prompt, true)
		if err != nil {
			errCh <- err
			return
		}

		resp, err := c.httpClient.Do(req)
		if err != nil {
			errCh <- fmt.Errorf("send stream request: %w", err)
			return
		}
		defer func() {
			_ = resp.Body.Close()
		}()

		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			body, readErr := io.ReadAll(resp.Body)
			if readErr != nil {
				errCh <- fmt.Errorf("ollama status %d: read error body: %w", resp.StatusCode, readErr)
				return
			}
			errCh <- &APIError{
				Backend: "ollama",
				Type:    fmt.Sprintf("status_%d", resp.StatusCode),
				Message: strings.TrimSpace(string(body)),
			}
			return
		}

		scanner := bufio.NewScanner(resp.Body)
		for scanner.Scan() {
			var chunk ollamaResponse
			if err := json.Unmarshal(scanner.Bytes(), &chunk); err != nil {
				errCh <- fmt.Errorf("decode stream chunk: %w", err)
				return
			}
			if strings.TrimSpace(chunk.Error) != "" {
				errCh <- &APIError{Backend: "ollama", Type: "request_failed", Message: chunk.Error}
				return
			}

			if chunk.Message.Content != "" {
				select {
				case tokenCh <- chunk.Message.Content:
				case <-ctx.Done():
					return
				}
			}

			if chunk.Done {
				return
			}
		}

		if err := scanner.Err(); err != nil && !errors.Is(err, context.Canceled) {
			errCh <- fmt.Errorf("read stream response: %w", err)
		}
	}()

	return StreamResult{Tokens: tokenCh, Err: errCh}
}

func (c *ollamaClient) newRequest(ctx context.Context, system, prompt string, stream bool) (*http.Request, error) {
	messages := make([]apiMessage, 0, 2)
	if strings.TrimSpace(system) != "" {
		messages = append(messages, apiMessage{Role: "system", Content: system})
	}
	messages = append(messages, apiMessage{Role: "user", Content: prompt})

	payload, err := json.Marshal(ollamaRequest{
		Model:    c.model,
		Messages: messages,
		Stream:   stream,
	})
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/api/chat", bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	return req, nil
}
