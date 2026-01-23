// Package autoupdate provides rate limiting for LLM and HTTP requests.
package autoupdate

import (
	"context"
	"errors"
	"net/url"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

// Error variables for rate limiting errors
var (
	// ErrRateLimitExceeded is returned when a rate limit is exceeded and context is cancelled
	ErrRateLimitExceeded = errors.New("rate limit exceeded")
)

// RateLimiter manages request rate limiting for LLM and HTTP requests.
// It enforces:
// - LLM rate limiting: 5 requests per minute
// - HTTP rate limiting: 10 requests per minute per domain
type RateLimiter struct {
	// llmLimiter limits LLM API requests to 5 per minute
	llmLimiter *rate.Limiter
	// httpLimiters maps domain names to their rate limiters (10 per minute per domain)
	httpLimiters map[string]*rate.Limiter
	// mu protects httpLimiters map
	mu sync.Mutex
	// clock allows overriding time functions for testing
	clock Clock
}

// Clock interface allows mocking time for testing
type Clock interface {
	Now() time.Time
	Sleep(d time.Duration)
}

// realClock implements Clock using actual time functions
type realClock struct{}

func (realClock) Now() time.Time         { return time.Now() }
func (realClock) Sleep(d time.Duration)  { time.Sleep(d) }

// RateLimiterOption configures a RateLimiter
type RateLimiterOption func(*RateLimiter)

// WithClock sets a custom clock for testing
func WithClock(clock Clock) RateLimiterOption {
	return func(r *RateLimiter) {
		r.clock = clock
	}
}

// NewRateLimiter creates a new rate limiter with default settings.
// LLM requests are limited to 5 per minute.
// HTTP requests are limited to 10 per minute per domain.
func NewRateLimiter(opts ...RateLimiterOption) *RateLimiter {
	r := &RateLimiter{
		// 5 requests per minute = 5/60 = 1 request per 12 seconds
		// Allow burst of 1 to ensure strict rate limiting
		llmLimiter:   rate.NewLimiter(rate.Every(12*time.Second), 1),
		httpLimiters: make(map[string]*rate.Limiter),
		clock:        realClock{},
	}

	for _, opt := range opts {
		opt(r)
	}

	return r
}

// WaitLLM waits for LLM rate limit before proceeding.
// It blocks until a token is available or the context is cancelled.
// Returns ErrRateLimitExceeded if the context is cancelled while waiting.
func (r *RateLimiter) WaitLLM(ctx context.Context) error {
	err := r.llmLimiter.Wait(ctx)
	if err != nil {
		// Check for context cancellation or deadline exceeded
		if ctx.Err() != nil {
			return ErrRateLimitExceeded
		}
		// For other errors (like burst exceeded), wrap them
		return err
	}
	return nil
}

// WaitHTTP waits for HTTP rate limit for a specific domain before proceeding.
// It blocks until a token is available or the context is cancelled.
// Returns ErrRateLimitExceeded if the context is cancelled while waiting.
func (r *RateLimiter) WaitHTTP(ctx context.Context, domain string) error {
	limiter := r.getHTTPLimiter(domain)
	err := limiter.Wait(ctx)
	if err != nil {
		// Check for context cancellation or deadline exceeded
		if ctx.Err() != nil {
			return ErrRateLimitExceeded
		}
		// For other errors (like burst exceeded), wrap them
		return err
	}
	return nil
}

// WaitHTTPForURL waits for HTTP rate limit for a URL's domain before proceeding.
// It extracts the domain from the URL and applies rate limiting.
func (r *RateLimiter) WaitHTTPForURL(ctx context.Context, rawURL string) error {
	domain, err := extractDomain(rawURL)
	if err != nil {
		// If we can't parse the URL, use the raw URL as the domain
		domain = rawURL
	}
	return r.WaitHTTP(ctx, domain)
}

// getHTTPLimiter returns the rate limiter for a specific domain.
// Creates a new limiter if one doesn't exist for the domain.
func (r *RateLimiter) getHTTPLimiter(domain string) *rate.Limiter {
	r.mu.Lock()
	defer r.mu.Unlock()

	limiter, exists := r.httpLimiters[domain]
	if !exists {
		// 10 requests per minute = 10/60 = 1 request per 6 seconds
		// Allow burst of 1 to ensure strict rate limiting
		limiter = rate.NewLimiter(rate.Every(6*time.Second), 1)
		r.httpLimiters[domain] = limiter
	}
	return limiter
}

// AllowLLM reports whether an LLM request may happen now.
// It does not block or consume a token.
func (r *RateLimiter) AllowLLM() bool {
	return r.llmLimiter.Allow()
}

// AllowHTTP reports whether an HTTP request to the domain may happen now.
// It does not block or consume a token.
func (r *RateLimiter) AllowHTTP(domain string) bool {
	limiter := r.getHTTPLimiter(domain)
	return limiter.Allow()
}

// ReserveLLM returns a Reservation that indicates how long the caller must wait
// before an LLM request can proceed.
func (r *RateLimiter) ReserveLLM() *rate.Reservation {
	return r.llmLimiter.Reserve()
}

// ReserveHTTP returns a Reservation that indicates how long the caller must wait
// before an HTTP request to the domain can proceed.
func (r *RateLimiter) ReserveHTTP(domain string) *rate.Reservation {
	limiter := r.getHTTPLimiter(domain)
	return limiter.Reserve()
}

// LLMLimit returns the current LLM rate limit (requests per second).
func (r *RateLimiter) LLMLimit() rate.Limit {
	return r.llmLimiter.Limit()
}

// HTTPLimit returns the current HTTP rate limit for a domain (requests per second).
func (r *RateLimiter) HTTPLimit(domain string) rate.Limit {
	limiter := r.getHTTPLimiter(domain)
	return limiter.Limit()
}

// SetLLMLimit sets a custom LLM rate limit (for testing).
func (r *RateLimiter) SetLLMLimit(limit rate.Limit, burst int) {
	r.llmLimiter.SetLimit(limit)
	r.llmLimiter.SetBurst(burst)
}

// SetHTTPLimit sets a custom HTTP rate limit for a domain (for testing).
func (r *RateLimiter) SetHTTPLimit(domain string, limit rate.Limit, burst int) {
	limiter := r.getHTTPLimiter(domain)
	limiter.SetLimit(limit)
	limiter.SetBurst(burst)
}

// extractDomain extracts the domain from a URL string.
func extractDomain(rawURL string) (string, error) {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return "", err
	}
	return parsed.Host, nil
}

// DomainCount returns the number of domains being tracked for HTTP rate limiting.
func (r *RateLimiter) DomainCount() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.httpLimiters)
}

// Reset clears all HTTP domain limiters and resets the LLM limiter.
// Useful for testing.
func (r *RateLimiter) Reset() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.httpLimiters = make(map[string]*rate.Limiter)
	r.llmLimiter = rate.NewLimiter(rate.Every(12*time.Second), 1)
}
