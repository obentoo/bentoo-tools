package autoupdate

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

// =============================================================================
// Unit Tests
// =============================================================================

// TestNewLLMClientMissingProvider tests that NewLLMClient returns error when provider is empty
func TestNewLLMClientMissingProvider(t *testing.T) {
	cfg := LLMConfig{
		Provider:  "",
		APIKeyEnv: "TEST_API_KEY",
		Model:     "claude-3-haiku-20240307",
	}

	_, err := NewLLMClient(cfg)
	if err == nil {
		t.Error("Expected error for missing provider")
	}
	if err != ErrLLMNotConfigured {
		t.Errorf("Expected ErrLLMNotConfigured, got: %v", err)
	}
}

// TestNewLLMClientUnsupportedProvider tests that NewLLMClient returns error for unsupported provider
func TestNewLLMClientUnsupportedProvider(t *testing.T) {
	cfg := LLMConfig{
		Provider:  "openai",
		APIKeyEnv: "TEST_API_KEY",
		Model:     "gpt-4",
	}

	_, err := NewLLMClient(cfg)
	if err == nil {
		t.Error("Expected error for unsupported provider")
	}
}

// TestNewLLMClientMissingAPIKeyEnv tests that NewLLMClient returns error when api_key_env is empty
func TestNewLLMClientMissingAPIKeyEnv(t *testing.T) {
	cfg := LLMConfig{
		Provider:  "claude",
		APIKeyEnv: "",
		Model:     "claude-3-haiku-20240307",
	}

	_, err := NewLLMClient(cfg)
	if err == nil {
		t.Error("Expected error for missing api_key_env")
	}
}

// TestNewLLMClientMissingAPIKey tests that NewLLMClient returns error when API key env var is not set
func TestNewLLMClientMissingAPIKey(t *testing.T) {
	// Ensure the env var is not set
	os.Unsetenv("TEST_MISSING_API_KEY")

	cfg := LLMConfig{
		Provider:  "claude",
		APIKeyEnv: "TEST_MISSING_API_KEY",
		Model:     "claude-3-haiku-20240307",
	}

	_, err := NewLLMClient(cfg)
	if err == nil {
		t.Error("Expected error for missing API key")
	}
}

// TestNewLLMClientSuccess tests successful LLM client creation
func TestNewLLMClientSuccess(t *testing.T) {
	// Set up test API key
	os.Setenv("TEST_LLM_API_KEY", "test-key-12345")
	defer os.Unsetenv("TEST_LLM_API_KEY")

	cfg := LLMConfig{
		Provider:  "claude",
		APIKeyEnv: "TEST_LLM_API_KEY",
		Model:     "claude-3-haiku-20240307",
	}

	client, err := NewLLMClient(cfg)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if client == nil {
		t.Fatal("Expected non-nil client")
	}
}

// TestNewLLMClientDefaultModel tests that default model is set when not specified
func TestNewLLMClientDefaultModel(t *testing.T) {
	os.Setenv("TEST_LLM_API_KEY", "test-key-12345")
	defer os.Unsetenv("TEST_LLM_API_KEY")

	cfg := LLMConfig{
		Provider:  "claude",
		APIKeyEnv: "TEST_LLM_API_KEY",
		Model:     "", // Empty model
	}

	client, err := NewLLMClient(cfg)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if client.config.Model != "claude-3-haiku-20240307" {
		t.Errorf("Expected default model 'claude-3-haiku-20240307', got %q", client.config.Model)
	}
}

// TestExtractVersionClaudeSuccess tests successful version extraction with mocked Claude API
func TestExtractVersionClaudeSuccess(t *testing.T) {
	// Create mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request method
		if r.Method != "POST" {
			t.Errorf("Expected POST, got %s", r.Method)
		}

		// Verify headers
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("Expected Content-Type application/json, got %s", r.Header.Get("Content-Type"))
		}
		if r.Header.Get("x-api-key") != "test-key-12345" {
			t.Errorf("Expected x-api-key test-key-12345, got %s", r.Header.Get("x-api-key"))
		}
		if r.Header.Get("anthropic-version") != "2023-06-01" {
			t.Errorf("Expected anthropic-version 2023-06-01, got %s", r.Header.Get("anthropic-version"))
		}

		// Return mock response
		resp := claudeResponse{
			ID:   "msg_test123",
			Type: "message",
			Role: "assistant",
			Content: []contentBlock{
				{Type: "text", Text: "11.81.1"},
			},
			Model:      "claude-3-haiku-20240307",
			StopReason: "end_turn",
			Usage:      claudeUsage{InputTokens: 100, OutputTokens: 10},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	// Set up test API key
	os.Setenv("TEST_LLM_API_KEY", "test-key-12345")
	defer os.Unsetenv("TEST_LLM_API_KEY")

	cfg := LLMConfig{
		Provider:  "claude",
		APIKeyEnv: "TEST_LLM_API_KEY",
		Model:     "claude-3-haiku-20240307",
	}

	client, err := NewLLMClient(cfg)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	// Override HTTP client to use mock server
	client.httpClient = &http.Client{
		Transport: &mockTransport{server: server},
	}

	content := []byte(`{"version": "11.81.1", "notes": [{"version": "11.81.1"}]}`)
	version, err := client.ExtractVersion(content, "Extract the version number")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if version != "11.81.1" {
		t.Errorf("Expected version '11.81.1', got %q", version)
	}
}

// mockTransport redirects requests to the test server
type mockTransport struct {
	server *httptest.Server
}

func (t *mockTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// Redirect to test server
	req.URL.Scheme = "http"
	req.URL.Host = t.server.Listener.Addr().String()
	return http.DefaultTransport.RoundTrip(req)
}

// TestExtractVersionClaudeAPIError tests handling of API errors
func TestExtractVersionClaudeAPIError(t *testing.T) {
	// Create mock server that returns an error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		resp := claudeErrorResponse{
			Type: "error",
			Error: struct {
				Type    string `json:"type"`
				Message string `json:"message"`
			}{
				Type:    "authentication_error",
				Message: "Invalid API key",
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	os.Setenv("TEST_LLM_API_KEY", "invalid-key")
	defer os.Unsetenv("TEST_LLM_API_KEY")

	cfg := LLMConfig{
		Provider:  "claude",
		APIKeyEnv: "TEST_LLM_API_KEY",
		Model:     "claude-3-haiku-20240307",
	}

	client, err := NewLLMClient(cfg)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	client.httpClient = &http.Client{
		Transport: &mockTransport{server: server},
	}

	_, err = client.ExtractVersion([]byte("test content"), "Extract version")
	if err == nil {
		t.Error("Expected error for API error response")
	}
}

// TestExtractVersionClaudeEmptyResponse tests handling of empty response
func TestExtractVersionClaudeEmptyResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := claudeResponse{
			ID:         "msg_test123",
			Type:       "message",
			Role:       "assistant",
			Content:    []contentBlock{}, // Empty content
			Model:      "claude-3-haiku-20240307",
			StopReason: "end_turn",
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	os.Setenv("TEST_LLM_API_KEY", "test-key")
	defer os.Unsetenv("TEST_LLM_API_KEY")

	cfg := LLMConfig{
		Provider:  "claude",
		APIKeyEnv: "TEST_LLM_API_KEY",
		Model:     "claude-3-haiku-20240307",
	}

	client, err := NewLLMClient(cfg)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	client.httpClient = &http.Client{
		Transport: &mockTransport{server: server},
	}

	_, err = client.ExtractVersion([]byte("test content"), "Extract version")
	if err != ErrLLMEmptyResponse {
		t.Errorf("Expected ErrLLMEmptyResponse, got: %v", err)
	}
}

// TestExtractVersionClaudeNetworkError tests handling of network errors
func TestExtractVersionClaudeNetworkError(t *testing.T) {
	os.Setenv("TEST_LLM_API_KEY", "test-key")
	defer os.Unsetenv("TEST_LLM_API_KEY")

	cfg := LLMConfig{
		Provider:  "claude",
		APIKeyEnv: "TEST_LLM_API_KEY",
		Model:     "claude-3-haiku-20240307",
	}

	client, err := NewLLMClient(cfg)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	// Use a transport that always fails
	client.httpClient = &http.Client{
		Transport: &failingTransport{},
	}

	_, err = client.ExtractVersion([]byte("test content"), "Extract version")
	if err == nil {
		t.Error("Expected error for network failure")
	}
}

// failingTransport always returns an error
type failingTransport struct{}

func (t *failingTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	return nil, &http.MaxBytesError{}
}

// TestCleanVersionString tests version string cleanup
func TestCleanVersionString(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"11.81.1", "11.81.1"},
		{"v11.81.1", "11.81.1"},
		{"V11.81.1", "11.81.1"},
		{" 11.81.1 ", "11.81.1"},
		{"\"11.81.1\"", "11.81.1"},
		{"'11.81.1'", "11.81.1"},
		{"`11.81.1`", "11.81.1"},
		{"11.81.1.", "11.81.1"},
		{"11.81.1,", "11.81.1"},
		{"  v11.81.1.  ", "11.81.1"},
	}

	for _, tc := range tests {
		result := cleanVersionString(tc.input)
		if result != tc.expected {
			t.Errorf("cleanVersionString(%q) = %q, expected %q", tc.input, result, tc.expected)
		}
	}
}

// TestBuildVersionExtractionPrompt tests prompt building
func TestBuildVersionExtractionPrompt(t *testing.T) {
	content := []byte(`{"version": "1.2.3"}`)
	userPrompt := "Extract the latest version"

	prompt := buildVersionExtractionPrompt(content, userPrompt)

	// Check that prompt contains expected elements
	if len(prompt) == 0 {
		t.Error("Expected non-empty prompt")
	}

	// Should contain the user prompt
	if !containsString(prompt, userPrompt) {
		t.Error("Prompt should contain user prompt")
	}

	// Should contain the content
	if !containsString(prompt, `{"version": "1.2.3"}`) {
		t.Error("Prompt should contain content")
	}

	// Should contain instructions for version-only response
	if !containsString(prompt, "ONLY the version number") {
		t.Error("Prompt should contain version-only instruction")
	}
}

// TestBuildVersionExtractionPromptTruncation tests content truncation
func TestBuildVersionExtractionPromptTruncation(t *testing.T) {
	// Create content larger than maxContentLen (4000)
	largeContent := make([]byte, 5000)
	for i := range largeContent {
		largeContent[i] = 'x'
	}

	prompt := buildVersionExtractionPrompt(largeContent, "")

	// Should contain truncation indicator
	if !containsString(prompt, "truncated") {
		t.Error("Prompt should indicate truncation for large content")
	}
}

// TestBuildVersionExtractionPromptEmptyUserPrompt tests prompt without user prompt
func TestBuildVersionExtractionPromptEmptyUserPrompt(t *testing.T) {
	content := []byte(`{"version": "1.2.3"}`)

	prompt := buildVersionExtractionPrompt(content, "")

	// Should not contain "Instructions:" when user prompt is empty
	if containsString(prompt, "Instructions:") {
		t.Error("Prompt should not contain Instructions when user prompt is empty")
	}
}

// TestExtractTextFromResponse tests text extraction from Claude response
func TestExtractTextFromResponse(t *testing.T) {
	tests := []struct {
		name     string
		resp     claudeResponse
		expected string
	}{
		{
			name: "single text block",
			resp: claudeResponse{
				Content: []contentBlock{
					{Type: "text", Text: "11.81.1"},
				},
			},
			expected: "11.81.1",
		},
		{
			name: "multiple blocks, first is text",
			resp: claudeResponse{
				Content: []contentBlock{
					{Type: "text", Text: "1.2.3"},
					{Type: "tool_use", Text: ""},
				},
			},
			expected: "1.2.3",
		},
		{
			name: "empty content",
			resp: claudeResponse{
				Content: []contentBlock{},
			},
			expected: "",
		},
		{
			name: "no text blocks",
			resp: claudeResponse{
				Content: []contentBlock{
					{Type: "tool_use", Text: ""},
				},
			},
			expected: "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := extractTextFromResponse(tc.resp)
			if result != tc.expected {
				t.Errorf("extractTextFromResponse() = %q, expected %q", result, tc.expected)
			}
		})
	}
}

// TestExtractVersionRequestFormat tests that the request is properly formatted
func TestExtractVersionRequestFormat(t *testing.T) {
	var capturedRequest claudeRequest

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Capture the request body
		json.NewDecoder(r.Body).Decode(&capturedRequest)

		// Return success response
		resp := claudeResponse{
			ID:   "msg_test",
			Type: "message",
			Role: "assistant",
			Content: []contentBlock{
				{Type: "text", Text: "1.0.0"},
			},
			StopReason: "end_turn",
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	os.Setenv("TEST_LLM_API_KEY", "test-key")
	defer os.Unsetenv("TEST_LLM_API_KEY")

	cfg := LLMConfig{
		Provider:  "claude",
		APIKeyEnv: "TEST_LLM_API_KEY",
		Model:     "claude-3-haiku-20240307",
	}

	client, err := NewLLMClient(cfg)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	client.httpClient = &http.Client{
		Transport: &mockTransport{server: server},
	}

	client.ExtractVersion([]byte("test content"), "Extract version")

	// Verify request format
	if capturedRequest.Model != "claude-3-haiku-20240307" {
		t.Errorf("Expected model 'claude-3-haiku-20240307', got %q", capturedRequest.Model)
	}
	if capturedRequest.MaxTokens != 100 {
		t.Errorf("Expected max_tokens 100, got %d", capturedRequest.MaxTokens)
	}
	if len(capturedRequest.Messages) != 1 {
		t.Errorf("Expected 1 message, got %d", len(capturedRequest.Messages))
	}
	if capturedRequest.Messages[0].Role != "user" {
		t.Errorf("Expected role 'user', got %q", capturedRequest.Messages[0].Role)
	}
}

// TestExtractVersionWithVersionPrefix tests that version prefixes are cleaned
func TestExtractVersionWithVersionPrefix(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := claudeResponse{
			ID:   "msg_test",
			Type: "message",
			Role: "assistant",
			Content: []contentBlock{
				{Type: "text", Text: "v1.2.3"}, // With 'v' prefix
			},
			StopReason: "end_turn",
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	os.Setenv("TEST_LLM_API_KEY", "test-key")
	defer os.Unsetenv("TEST_LLM_API_KEY")

	cfg := LLMConfig{
		Provider:  "claude",
		APIKeyEnv: "TEST_LLM_API_KEY",
		Model:     "claude-3-haiku-20240307",
	}

	client, err := NewLLMClient(cfg)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	client.httpClient = &http.Client{
		Transport: &mockTransport{server: server},
	}

	version, err := client.ExtractVersion([]byte("test"), "")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if version != "1.2.3" {
		t.Errorf("Expected version '1.2.3' (without prefix), got %q", version)
	}
}

// containsString checks if a string contains a substring
func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && findSubstring(s, substr)))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
