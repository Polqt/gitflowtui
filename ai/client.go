package ai

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

const (
	anthropicEndpoint = "https://api.anthropic.com/v1/messages"
	anthropicVersion  = "2023-06-01"
	// DefaultModel is the model used for all AI calls.
	// Pinned to Sonnet 4 for the best balance of speed and reasoning quality.
	DefaultModel     = "claude-sonnet-4-20250514"
	defaultMaxTokens = 2048
	// streamBufSize is the channel buffer for streaming tokens.
	// Large enough to absorb burst writes without blocking the HTTP goroutine.
	streamBufSize = 64
)

type apiMessage struct {
	Role    string `json:"role"` // "user" or "assistant"
	Content string `json:"content"`
}

type apiRequest struct {
	Model     string       `json:"model"`
	MaxTokens int          `json:"max_tokens"`
	System    string       `json:"system,omitempty"`
	Messages  []apiMessage `json:"messages"`
	Stream    bool         `json:"stream,omitempty"`
}

type apiResponse struct {
	Content []struct {
		Type string `json:"type"` // "message_start", "message_end", or "message_chunk"
		Text string `json:"text"` // Present if Type is "message_chunk"
	} `json:"content"`
	Error *struct {
		Type    string `json:"type"` // e.g. "invalid_request_error"
		Message string `json:"message"`
	} `json:"error"`
}

type client struct {
	apiKey     string
	model      string
	httpClient *http.Client
}

func newClient(apiKey, model string) *client {
	if strings.TrimSpace(apiKey) == "" {
		return nil
	}
	if model == "" {
		model = DefaultModel
	}
	return &client{
		apiKey: apiKey,
		model:  model,
		httpClient: &http.Client{
			Timeout: 90 * time.Second,
		},
	}
}

// complete sends a single-turn prompt and returns the full response text.
// It is used for structured-JSON responses where we need the complete output
// before parsing (commit suggestions, conflict analysis, health reports).
func (c *client) complete(ctx context.Context, system, prompt string, maxTokens int) (string, error) {
	if maxTokens <= 0 {
		maxTokens = defaultMaxTokens
	}

	payload, err := json.Marshal(apiRequest{
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
	defer resp.Body.Close()

	var result apiResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("decode response (status %d): %w", resp.StatusCode, err)
	}
	if result.Error != nil {
		return "", &ErrAPIFailure{Type: result.Error.Type, Message: result.Error.Message}
	}
	if len(result.Content) == 0 {
		return "", fmt.Errorf("empty content in API response")
	}
	return result.Content[0].Text, nil
}

// stream sends a prompt and returns channels for receiving tokens and errors as they arrive.
type StreamResult struct {
	// Tokens receives text tokens as they arrive from the API.
	// The channel is closed when the stream ends (success or error).
	Tokens <-chan string
	// Err receives at most one error, then is closed.
	// If nil is received the stream completed successfully.
	Err <-chan error
}

func (c *client) stream(ctx context.Context, system, prompt string, maxTokens int) StreamResult {
	tokenCh := make(chan string, streamBufSize)
	errCh := make(chan error, 1)

	if maxTokens <= 0 {
		maxTokens = defaultMaxTokens
	}

	go func() {
		defer close(tokenCh)
		defer close(errCh)

		payload, err := json.Marshal(apiRequest{
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

		// use a fresh client w/out timeout since we'll manage cancellation via ctx
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
		defer resp.Body.Close()

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

			// parse sse event, only content_block_delta with text delta carries tokens.
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
				errCh <- &ErrAPIFailure{Type: event.Error.Type, Message: event.Error.Message}
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
	}()

	return StreamResult{
		Tokens: tokenCh,
		Err:    errCh,
	}

}

func (c *client) setHeaders(req *http.Request) {
	req.Header.Set("x-api-key", c.apiKey)
	req.Header.Set("anthropic-version", anthropicVersion)
	req.Header.Set("content-type", "application/json")
}
