// Package autoupdate provides LLM integration for version extraction.
package autoupdate

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

// Error variables for LLM errors
var (
	// ErrLLMNotConfigured is returned when LLM provider is not configured
	ErrLLMNotConfigured = errors.New("LLM provider not configured")
	// ErrLLMAPIKeyMissing is returned when the API key environment variable is not set
	ErrLLMAPIKeyMissing = errors.New("LLM API key environment variable not set")
	// ErrLLMUnsupportedProvider is returned when an unsupported LLM provider is specified
	ErrLLMUnsupportedProvider = errors.New("unsupported LLM provider")
	// ErrLLMRequestFailed is returned when the LLM API request fails
	ErrLLMRequestFailed = errors.New("LLM API request failed")
	// ErrLLMEmptyResponse is returned when the LLM returns an empty response
	ErrLLMEmptyResponse = errors.New("LLM returned empty response")
)

// LLMConfig holds LLM provider configuration.
// It defines which LLM service to use and how to authenticate.
type LLMConfig struct {
	// Provider is the LLM provider name (e.g., "claude")
	Provider string
	// APIKeyEnv is the environment variable name containing the API key
	APIKeyEnv string
	// Model is the specific model to use (e.g., "claude-3-haiku-20240307")
	Model string
}

// LLMClient handles LLM API interactions for version extraction.
// It supports the Claude API for extracting version information from content.
type LLMClient struct {
	config     LLMConfig
	httpClient *http.Client
	apiKey     string
}

// claudeRequest represents the request body for Claude Messages API
type claudeRequest struct {
	Model     string          `json:"model"`
	MaxTokens int             `json:"max_tokens"`
	Messages  []claudeMessage `json:"messages"`
}

// claudeMessage represents a message in the Claude conversation
type claudeMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// claudeResponse represents the response from Claude Messages API
type claudeResponse struct {
	ID           string         `json:"id"`
	Type         string         `json:"type"`
	Role         string         `json:"role"`
	Content      []contentBlock `json:"content"`
	Model        string         `json:"model"`
	StopReason   string         `json:"stop_reason"`
	StopSequence *string        `json:"stop_sequence"`
	Usage        claudeUsage    `json:"usage"`
}

// contentBlock represents a content block in Claude's response
type contentBlock struct {
	Type string `json:"type"`
	Text string `json:"text,omitempty"`
}

// claudeUsage represents token usage information
type claudeUsage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}

// claudeErrorResponse represents an error response from Claude API
type claudeErrorResponse struct {
	Type  string `json:"type"`
	Error struct {
		Type    string `json:"type"`
		Message string `json:"message"`
	} `json:"error"`
}

// NewLLMClient creates a new LLM client from configuration.
// It validates the configuration and retrieves the API key from the environment.
// Returns an error if the provider is not configured or the API key is missing.
func NewLLMClient(cfg LLMConfig) (*LLMClient, error) {
	// Check if provider is configured
	if cfg.Provider == "" {
		return nil, ErrLLMNotConfigured
	}

	// Validate provider
	if cfg.Provider != "claude" {
		return nil, fmt.Errorf("%w: %s", ErrLLMUnsupportedProvider, cfg.Provider)
	}

	// Check API key environment variable name
	if cfg.APIKeyEnv == "" {
		return nil, fmt.Errorf("%w: api_key_env not specified", ErrLLMNotConfigured)
	}

	// Get API key from environment
	apiKey := os.Getenv(cfg.APIKeyEnv)
	if apiKey == "" {
		return nil, fmt.Errorf("%w: %s", ErrLLMAPIKeyMissing, cfg.APIKeyEnv)
	}

	// Set default model if not specified
	model := cfg.Model
	if model == "" {
		model = "claude-3-haiku-20240307"
	}

	return &LLMClient{
		config: LLMConfig{
			Provider:  cfg.Provider,
			APIKeyEnv: cfg.APIKeyEnv,
			Model:     model,
		},
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		apiKey: apiKey,
	}, nil
}

// NewLLMClientWithHTTPClient creates a new LLM client with a custom HTTP client.
// This is useful for testing with mock servers.
func NewLLMClientWithHTTPClient(cfg LLMConfig, httpClient *http.Client) (*LLMClient, error) {
	client, err := NewLLMClient(cfg)
	if err != nil {
		return nil, err
	}
	client.httpClient = httpClient
	return client, nil
}

// ExtractVersion uses the LLM to extract a version string from content.
// It sends the content along with the provided prompt to the LLM and
// parses the response to extract the version number.
func (c *LLMClient) ExtractVersion(content []byte, prompt string) (string, error) {
	if c.config.Provider != "claude" {
		return "", fmt.Errorf("%w: %s", ErrLLMUnsupportedProvider, c.config.Provider)
	}

	return c.extractVersionClaude(content, prompt)
}

// extractVersionClaude extracts version using Claude API
func (c *LLMClient) extractVersionClaude(content []byte, prompt string) (string, error) {
	// Build the user message with content and prompt
	userMessage := buildVersionExtractionPrompt(content, prompt)

	// Create request body
	reqBody := claudeRequest{
		Model:     c.config.Model,
		MaxTokens: 100, // Version extraction needs minimal tokens
		Messages: []claudeMessage{
			{
				Role:    "user",
				Content: userMessage,
			},
		},
	}

	// Marshal request body
	reqJSON, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request
	req, err := http.NewRequest("POST", "https://api.anthropic.com/v1/messages", bytes.NewReader(reqJSON))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	// Set required headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", c.apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	// Send request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrLLMRequestFailed, err)
	}
	defer resp.Body.Close()

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	// Check for error response
	if resp.StatusCode != http.StatusOK {
		var errResp claudeErrorResponse
		if err := json.Unmarshal(body, &errResp); err == nil && errResp.Error.Message != "" {
			return "", fmt.Errorf("%w: %s (status %d)", ErrLLMRequestFailed, errResp.Error.Message, resp.StatusCode)
		}
		return "", fmt.Errorf("%w: status %d", ErrLLMRequestFailed, resp.StatusCode)
	}

	// Parse response
	var claudeResp claudeResponse
	if err := json.Unmarshal(body, &claudeResp); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	// Extract text from response
	version := extractTextFromResponse(claudeResp)
	if version == "" {
		return "", ErrLLMEmptyResponse
	}

	// Clean up the version string
	version = cleanVersionString(version)
	if version == "" {
		return "", ErrLLMEmptyResponse
	}

	return version, nil
}

// buildVersionExtractionPrompt creates the prompt for version extraction
func buildVersionExtractionPrompt(content []byte, userPrompt string) string {
	// Truncate content if too long (to avoid token limits)
	contentStr := string(content)
	const maxContentLen = 4000
	if len(contentStr) > maxContentLen {
		contentStr = contentStr[:maxContentLen] + "\n... (truncated)"
	}

	// Build the prompt
	var sb strings.Builder
	sb.WriteString("Extract the version number from the following content.\n\n")

	if userPrompt != "" {
		sb.WriteString("Instructions: ")
		sb.WriteString(userPrompt)
		sb.WriteString("\n\n")
	}

	sb.WriteString("Content:\n```\n")
	sb.WriteString(contentStr)
	sb.WriteString("\n```\n\n")
	sb.WriteString("Respond with ONLY the version number (e.g., \"1.2.3\" or \"11.81.1\"). ")
	sb.WriteString("Do not include any other text, explanation, or formatting.")

	return sb.String()
}

// extractTextFromResponse extracts the text content from Claude's response
func extractTextFromResponse(resp claudeResponse) string {
	for _, block := range resp.Content {
		if block.Type == "text" && block.Text != "" {
			return block.Text
		}
	}
	return ""
}

// cleanVersionString cleans up the version string from LLM response
func cleanVersionString(version string) string {
	// Trim whitespace
	version = strings.TrimSpace(version)

	// Remove common prefixes
	version = strings.TrimPrefix(version, "v")
	version = strings.TrimPrefix(version, "V")

	// Remove quotes if present
	version = strings.Trim(version, "\"'`")

	// Remove any trailing punctuation
	version = strings.TrimRight(version, ".,;:")

	// Trim whitespace again
	version = strings.TrimSpace(version)

	return version
}

// SetHTTPClient sets a custom HTTP client (useful for testing)
func (c *LLMClient) SetHTTPClient(client *http.Client) {
	c.httpClient = client
}

// SetBaseURL is a no-op for production but allows tests to override the API URL
// This method exists for API compatibility but the URL is hardcoded for security
func (c *LLMClient) SetBaseURL(url string) {
	// No-op in production - URL is hardcoded for security
	// Tests should use httptest.Server and custom HTTP client
}
