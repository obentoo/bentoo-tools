package autoupdate

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
	"golang.org/x/time/rate"
)

// =============================================================================
// Unit Tests
// =============================================================================

// TestNewRateLimiter tests that NewRateLimiter creates a valid rate limiter
func TestNewRateLimiter(t *testing.T) {
	rl := NewRateLimiter()
	if rl == nil {
		t.Fatal("Expected non-nil rate limiter")
	}
	if rl.llmLimiter == nil {
		t.Error("Expected non-nil LLM limiter")
	}
	if rl.httpLimiters == nil {
		t.Error("Expected non-nil HTTP limiters map")
	}
}

// TestLLMLimitValue tests that LLM limit is set to 5 per minute
func TestLLMLimitValue(t *testing.T) {
	rl := NewRateLimiter()
	// 5 per minute = 1 per 12 seconds = 1/12 per second
	expectedLimit := rate.Every(12 * time.Second)
	if rl.LLMLimit() != expectedLimit {
		t.Errorf("Expected LLM limit %v, got %v", expectedLimit, rl.LLMLimit())
	}
}

// TestHTTPLimitValue tests that HTTP limit is set to 10 per minute per domain
func TestHTTPLimitValue(t *testing.T) {
	rl := NewRateLimiter()
	// 10 per minute = 1 per 6 seconds = 1/6 per second
	expectedLimit := rate.Every(6 * time.Second)
	if rl.HTTPLimit("example.com") != expectedLimit {
		t.Errorf("Expected HTTP limit %v, got %v", expectedLimit, rl.HTTPLimit("example.com"))
	}
}


// TestHTTPLimiterPerDomain tests that each domain gets its own limiter
func TestHTTPLimiterPerDomain(t *testing.T) {
	rl := NewRateLimiter()

	// Access different domains
	_ = rl.HTTPLimit("example.com")
	_ = rl.HTTPLimit("github.com")
	_ = rl.HTTPLimit("pypi.org")

	if rl.DomainCount() != 3 {
		t.Errorf("Expected 3 domains, got %d", rl.DomainCount())
	}
}

// TestWaitLLMContextCancellation tests that WaitLLM respects context cancellation
func TestWaitLLMContextCancellation(t *testing.T) {
	rl := NewRateLimiter()
	// Consume the burst token first
	_ = rl.AllowLLM()

	// Now create a context that's already cancelled
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	err := rl.WaitLLM(ctx)
	if err != ErrRateLimitExceeded {
		t.Errorf("Expected ErrRateLimitExceeded, got %v", err)
	}
}

// TestWaitHTTPContextCancellation tests that WaitHTTP respects context cancellation
func TestWaitHTTPContextCancellation(t *testing.T) {
	rl := NewRateLimiter()
	domain := "example.com"
	// Consume the burst token first
	_ = rl.AllowHTTP(domain)

	// Now create a context that's already cancelled
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	err := rl.WaitHTTP(ctx, domain)
	if err != ErrRateLimitExceeded {
		t.Errorf("Expected ErrRateLimitExceeded, got %v", err)
	}
}

// TestWaitHTTPForURL tests URL domain extraction
func TestWaitHTTPForURL(t *testing.T) {
	rl := NewRateLimiter()

	ctx := context.Background()
	err := rl.WaitHTTPForURL(ctx, "https://api.github.com/repos/test/test")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	// Should have created a limiter for api.github.com
	if rl.DomainCount() != 1 {
		t.Errorf("Expected 1 domain, got %d", rl.DomainCount())
	}
}

// TestReset tests that Reset clears all state
func TestReset(t *testing.T) {
	rl := NewRateLimiter()

	// Create some HTTP limiters
	_ = rl.HTTPLimit("example.com")
	_ = rl.HTTPLimit("github.com")

	if rl.DomainCount() != 2 {
		t.Errorf("Expected 2 domains before reset, got %d", rl.DomainCount())
	}

	rl.Reset()

	if rl.DomainCount() != 0 {
		t.Errorf("Expected 0 domains after reset, got %d", rl.DomainCount())
	}
}

// TestAllowLLM tests the non-blocking AllowLLM method
func TestAllowLLM(t *testing.T) {
	rl := NewRateLimiter()
	// First request should be allowed (burst of 1)
	if !rl.AllowLLM() {
		t.Error("First LLM request should be allowed")
	}
	// Second immediate request should not be allowed
	if rl.AllowLLM() {
		t.Error("Second immediate LLM request should not be allowed")
	}
}

// TestAllowHTTP tests the non-blocking AllowHTTP method
func TestAllowHTTP(t *testing.T) {
	rl := NewRateLimiter()
	domain := "example.com"
	// First request should be allowed (burst of 1)
	if !rl.AllowHTTP(domain) {
		t.Error("First HTTP request should be allowed")
	}
	// Second immediate request should not be allowed
	if rl.AllowHTTP(domain) {
		t.Error("Second immediate HTTP request should not be allowed")
	}
}


// =============================================================================
// Property-Based Tests
// =============================================================================

// TestLLMRateLimiting tests Property 26: LLM Rate Limiting
// **Feature: autoupdate-analyzer, Property 26: LLM Rate Limiting**
// **Validates: Requirements 11.1**
//
// For any sequence of LLM requests, the rate limiter SHALL ensure no more than
// 5 requests are made within any 60-second window.
func TestLLMRateLimiting(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100
	properties := gopter.NewProperties(parameters)

	// Property: LLM rate is limited to 5 per minute (1 per 12 seconds)
	properties.Property("LLM rate limit is 5 per minute", prop.ForAll(
		func(dummy int) bool {
			rl := NewRateLimiter()
			// 5 per minute = 1 per 12 seconds
			expectedLimit := rate.Every(12 * time.Second)
			return rl.LLMLimit() == expectedLimit
		},
		gen.IntRange(1, 100),
	))

	// Property: First LLM request is always allowed (burst of 1)
	properties.Property("First LLM request is always allowed", prop.ForAll(
		func(dummy int) bool {
			rl := NewRateLimiter()
			return rl.AllowLLM()
		},
		gen.IntRange(1, 100),
	))

	// Property: Second immediate LLM request is not allowed
	properties.Property("Second immediate LLM request is not allowed", prop.ForAll(
		func(dummy int) bool {
			rl := NewRateLimiter()
			_ = rl.AllowLLM() // First request
			return !rl.AllowLLM() // Second should be denied
		},
		gen.IntRange(1, 100),
	))

	// Property: LLM reservation delay is approximately 12 seconds
	properties.Property("LLM reservation delay is approximately 12 seconds", prop.ForAll(
		func(dummy int) bool {
			rl := NewRateLimiter()
			_ = rl.AllowLLM() // Consume the burst token

			reservation := rl.ReserveLLM()
			delay := reservation.Delay()
			reservation.Cancel()

			// Delay should be close to 12 seconds (allow some tolerance)
			return delay >= 11*time.Second && delay <= 13*time.Second
		},
		gen.IntRange(1, 100),
	))

	// Property: Multiple LLM requests require waiting
	properties.Property("Multiple LLM requests require waiting", prop.ForAll(
		func(numRequests int) bool {
			if numRequests < 2 {
				return true // Skip trivial cases
			}

			rl := NewRateLimiter()
			allowedCount := 0

			for i := 0; i < numRequests; i++ {
				if rl.AllowLLM() {
					allowedCount++
				}
			}

			// Only 1 request should be allowed immediately (burst)
			return allowedCount == 1
		},
		gen.IntRange(2, 10),
	))

	// Property: WaitLLM respects context cancellation
	properties.Property("WaitLLM respects context cancellation", prop.ForAll(
		func(dummy int) bool {
			rl := NewRateLimiter()
			// Consume the burst token first
			_ = rl.AllowLLM()

			ctx, cancel := context.WithCancel(context.Background())
			cancel() // Cancel immediately

			err := rl.WaitLLM(ctx)
			return err == ErrRateLimitExceeded
		},
		gen.IntRange(1, 100),
	))

	// Property: Concurrent LLM requests are properly rate limited
	properties.Property("Concurrent LLM requests are properly rate limited", prop.ForAll(
		func(numGoroutines int) bool {
			if numGoroutines < 2 {
				return true
			}

			rl := NewRateLimiter()
			var allowedCount int
			var mu sync.Mutex
			var wg sync.WaitGroup

			for i := 0; i < numGoroutines; i++ {
				wg.Add(1)
				go func() {
					defer wg.Done()
					if rl.AllowLLM() {
						mu.Lock()
						allowedCount++
						mu.Unlock()
					}
				}()
			}

			wg.Wait()

			// Only 1 request should be allowed immediately
			return allowedCount == 1
		},
		gen.IntRange(2, 20),
	))

	properties.TestingRun(t)
}


// TestHTTPRateLimiting tests Property 27: HTTP Rate Limiting
// **Feature: autoupdate-analyzer, Property 27: HTTP Rate Limiting**
// **Validates: Requirements 11.2**
//
// For any sequence of HTTP requests to the same domain, the rate limiter SHALL
// ensure no more than 10 requests are made within any 60-second window.
func TestHTTPRateLimiting(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100
	properties := gopter.NewProperties(parameters)

	// Property: HTTP rate is limited to 10 per minute (1 per 6 seconds)
	properties.Property("HTTP rate limit is 10 per minute per domain", prop.ForAll(
		func(domain string) bool {
			rl := NewRateLimiter()
			// 10 per minute = 1 per 6 seconds
			expectedLimit := rate.Every(6 * time.Second)
			return rl.HTTPLimit(domain) == expectedLimit
		},
		gen.OneConstOf(
			"example.com",
			"github.com",
			"pypi.org",
			"api.github.com",
		),
	))

	// Property: First HTTP request to a domain is always allowed (burst of 1)
	properties.Property("First HTTP request to a domain is always allowed", prop.ForAll(
		func(domain string) bool {
			rl := NewRateLimiter()
			return rl.AllowHTTP(domain)
		},
		gen.OneConstOf(
			"example.com",
			"github.com",
			"pypi.org",
			"npmjs.com",
		),
	))

	// Property: Second immediate HTTP request to same domain is not allowed
	properties.Property("Second immediate HTTP request to same domain is not allowed", prop.ForAll(
		func(domain string) bool {
			rl := NewRateLimiter()
			_ = rl.AllowHTTP(domain) // First request
			return !rl.AllowHTTP(domain) // Second should be denied
		},
		gen.OneConstOf(
			"example.com",
			"github.com",
			"pypi.org",
			"crates.io",
		),
	))

	// Property: HTTP reservation delay is approximately 6 seconds
	properties.Property("HTTP reservation delay is approximately 6 seconds", prop.ForAll(
		func(domain string) bool {
			rl := NewRateLimiter()
			_ = rl.AllowHTTP(domain) // Consume the burst token

			reservation := rl.ReserveHTTP(domain)
			delay := reservation.Delay()
			reservation.Cancel()

			// Delay should be close to 6 seconds (allow some tolerance)
			return delay >= 5*time.Second && delay <= 7*time.Second
		},
		gen.OneConstOf(
			"example.com",
			"github.com",
			"pypi.org",
		),
	))

	// Property: Different domains have independent rate limits
	properties.Property("Different domains have independent rate limits", prop.ForAll(
		func(domain1, domain2 string) bool {
			if domain1 == domain2 {
				return true // Skip when domains are the same
			}

			rl := NewRateLimiter()

			// First request to domain1 should be allowed
			allowed1 := rl.AllowHTTP(domain1)
			// First request to domain2 should also be allowed (independent)
			allowed2 := rl.AllowHTTP(domain2)

			return allowed1 && allowed2
		},
		gen.OneConstOf("example.com", "github.com"),
		gen.OneConstOf("pypi.org", "npmjs.com"),
	))

	// Property: Multiple HTTP requests to same domain require waiting
	properties.Property("Multiple HTTP requests to same domain require waiting", prop.ForAll(
		func(numRequests int, domain string) bool {
			if numRequests < 2 {
				return true // Skip trivial cases
			}

			rl := NewRateLimiter()
			allowedCount := 0

			for i := 0; i < numRequests; i++ {
				if rl.AllowHTTP(domain) {
					allowedCount++
				}
			}

			// Only 1 request should be allowed immediately (burst)
			return allowedCount == 1
		},
		gen.IntRange(2, 10),
		gen.OneConstOf("example.com", "github.com", "pypi.org"),
	))

	// Property: WaitHTTP respects context cancellation
	properties.Property("WaitHTTP respects context cancellation", prop.ForAll(
		func(domain string) bool {
			rl := NewRateLimiter()
			// Consume the burst token first
			_ = rl.AllowHTTP(domain)

			ctx, cancel := context.WithCancel(context.Background())
			cancel() // Cancel immediately

			err := rl.WaitHTTP(ctx, domain)
			return err == ErrRateLimitExceeded
		},
		gen.OneConstOf(
			"example.com",
			"github.com",
			"pypi.org",
		),
	))

	// Property: Each domain gets its own limiter
	properties.Property("Each domain gets its own limiter", prop.ForAll(
		func(domains []string) bool {
			if len(domains) == 0 {
				return true
			}

			rl := NewRateLimiter()

			// Access each domain
			uniqueDomains := make(map[string]bool)
			for _, domain := range domains {
				_ = rl.HTTPLimit(domain)
				uniqueDomains[domain] = true
			}

			// Domain count should match unique domains
			return rl.DomainCount() == len(uniqueDomains)
		},
		gen.SliceOfN(5, gen.OneConstOf(
			"example.com",
			"github.com",
			"pypi.org",
			"npmjs.com",
			"crates.io",
		)),
	))

	// Property: Concurrent HTTP requests to same domain are properly rate limited
	properties.Property("Concurrent HTTP requests to same domain are properly rate limited", prop.ForAll(
		func(numGoroutines int, domain string) bool {
			if numGoroutines < 2 {
				return true
			}

			rl := NewRateLimiter()
			var allowedCount int
			var mu sync.Mutex
			var wg sync.WaitGroup

			for i := 0; i < numGoroutines; i++ {
				wg.Add(1)
				go func() {
					defer wg.Done()
					if rl.AllowHTTP(domain) {
						mu.Lock()
						allowedCount++
						mu.Unlock()
					}
				}()
			}

			wg.Wait()

			// Only 1 request should be allowed immediately
			return allowedCount == 1
		},
		gen.IntRange(2, 20),
		gen.OneConstOf("example.com", "github.com"),
	))

	// Property: URL domain extraction works correctly
	properties.Property("URL domain extraction works correctly", prop.ForAll(
		func(url string) bool {
			rl := NewRateLimiter()
			ctx := context.Background()

			// Should not error on valid URLs
			err := rl.WaitHTTPForURL(ctx, url)
			return err == nil
		},
		gen.OneConstOf(
			"https://api.github.com/repos/test/test",
			"https://pypi.org/pypi/requests/json",
			"https://registry.npmjs.org/express",
			"https://crates.io/api/v1/crates/serde",
		),
	))

	properties.TestingRun(t)
}
