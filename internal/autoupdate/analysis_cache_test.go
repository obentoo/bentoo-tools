package autoupdate

import (
	"testing"
	"time"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
)

// =============================================================================
// Property-Based Tests
// =============================================================================

// TestAnalysisCacheTTL tests Property 24: Analysis Cache TTL
// **Feature: autoupdate-analyzer, Property 24: Analysis Cache TTL**
// **Validates: Requirements 10.1**
func TestAnalysisCacheTTL(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100
	properties := gopter.NewProperties(parameters)

	// Property: Cache returns schema when within 24-hour TTL
	properties.Property("Cache returns schema when timestamp is within 24-hour TTL", prop.ForAll(
		func(pkg, url string, ageHours int) bool {
			tmpDir := t.TempDir()

			// Age must be positive and less than 24 hours
			if ageHours < 0 {
				ageHours = -ageHours
			}
			ageHours = ageHours % 23 // Ensure within TTL (0-22 hours)

			// Create a fixed "now" time
			fixedNow := time.Date(2026, 1, 23, 12, 0, 0, 0, time.UTC)
			entryTime := fixedNow.Add(-time.Duration(ageHours) * time.Hour)

			cache, err := NewAnalysisCache(tmpDir, WithAnalysisCacheNowFunc(func() time.Time { return fixedNow }))
			if err != nil {
				t.Logf("Failed to create analysis cache: %v", err)
				return false
			}

			// Create a test schema
			schema := &PackageConfig{
				URL:    url,
				Parser: "json",
				Path:   "version",
			}

			// Manually set entry with specific timestamp
			cache.Entries[pkg] = AnalysisCacheEntry{
				Schema:    schema,
				Timestamp: entryTime,
				URL:       url,
			}

			// Get should return the schema since it's within TTL
			result, found := cache.Get(pkg)
			if !found {
				t.Logf("Expected cache hit for age %d hours", ageHours)
				return false
			}
			return result.URL == url && result.Parser == "json"
		},
		genPackageName(),
		genValidURL(),
		gen.IntRange(0, 22),
	))

	// Property: Cache returns miss when 24-hour TTL expired
	properties.Property("Cache returns miss when timestamp exceeds 24-hour TTL", prop.ForAll(
		func(pkg, url string, extraHours int) bool {
			tmpDir := t.TempDir()

			// Extra hours beyond TTL (1-100 hours past expiry)
			if extraHours < 1 {
				extraHours = 1
			}
			extraHours = (extraHours % 100) + 1

			// Create a fixed "now" time
			fixedNow := time.Date(2026, 1, 23, 12, 0, 0, 0, time.UTC)
			// Entry time is 24 hours + extra hours ago (expired)
			entryTime := fixedNow.Add(-DefaultAnalysisCacheTTL - time.Duration(extraHours)*time.Hour)

			cache, err := NewAnalysisCache(tmpDir, WithAnalysisCacheNowFunc(func() time.Time { return fixedNow }))
			if err != nil {
				t.Logf("Failed to create analysis cache: %v", err)
				return false
			}

			// Create a test schema
			schema := &PackageConfig{
				URL:    url,
				Parser: "json",
				Path:   "version",
			}

			// Manually set entry with expired timestamp
			cache.Entries[pkg] = AnalysisCacheEntry{
				Schema:    schema,
				Timestamp: entryTime,
				URL:       url,
			}

			// Get should return miss since TTL expired
			_, found := cache.Get(pkg)
			if found {
				t.Logf("Expected cache miss for expired entry (extra %d hours past TTL)", extraHours)
				return false
			}
			return true
		},
		genPackageName(),
		genValidURL(),
		gen.IntRange(1, 100),
	))

	// Property: Entry at exactly 24 hours is considered expired
	properties.Property("Entry at exactly 24 hours is considered expired", prop.ForAll(
		func(pkg, url string) bool {
			tmpDir := t.TempDir()

			fixedNow := time.Date(2026, 1, 23, 12, 0, 0, 0, time.UTC)
			// Entry time is exactly 24 hours ago
			entryTime := fixedNow.Add(-DefaultAnalysisCacheTTL)

			cache, err := NewAnalysisCache(tmpDir, WithAnalysisCacheNowFunc(func() time.Time { return fixedNow }))
			if err != nil {
				t.Logf("Failed to create analysis cache: %v", err)
				return false
			}

			schema := &PackageConfig{
				URL:    url,
				Parser: "json",
				Path:   "version",
			}

			cache.Entries[pkg] = AnalysisCacheEntry{
				Schema:    schema,
				Timestamp: entryTime,
				URL:       url,
			}

			// Get should return miss since TTL is exactly reached
			_, found := cache.Get(pkg)
			return !found
		},
		genPackageName(),
		genValidURL(),
	))

	properties.TestingRun(t)
}


// TestCacheBypass tests Property 25: Cache Bypass
// **Feature: autoupdate-analyzer, Property 25: Cache Bypass**
// **Validates: Requirements 10.3**
func TestCacheBypass(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100
	properties := gopter.NewProperties(parameters)

	// Property: Bypass flag always returns cache miss regardless of TTL
	properties.Property("Bypass flag always returns cache miss regardless of TTL", prop.ForAll(
		func(pkg, url string) bool {
			tmpDir := t.TempDir()

			fixedNow := time.Date(2026, 1, 23, 12, 0, 0, 0, time.UTC)

			cache, err := NewAnalysisCache(tmpDir, WithAnalysisCacheNowFunc(func() time.Time { return fixedNow }))
			if err != nil {
				t.Logf("Failed to create analysis cache: %v", err)
				return false
			}

			// Create a test schema
			schema := &PackageConfig{
				URL:    url,
				Parser: "json",
				Path:   "version",
			}

			// Set a fresh entry (just now)
			cache.Entries[pkg] = AnalysisCacheEntry{
				Schema:    schema,
				Timestamp: fixedNow,
				URL:       url,
			}

			// GetWithBypass(bypass=true) should return miss
			_, found := cache.GetWithBypass(pkg, true)
			if found {
				t.Log("Expected cache miss when bypass=true")
				return false
			}

			// GetWithBypass(bypass=false) should return hit
			result, found := cache.GetWithBypass(pkg, false)
			if !found {
				t.Log("Expected cache hit when bypass=false")
				return false
			}
			return result.URL == url
		},
		genPackageName(),
		genValidURL(),
	))

	// Property: Bypass does not affect cache contents
	properties.Property("Bypass does not affect cache contents", prop.ForAll(
		func(pkg, url string) bool {
			tmpDir := t.TempDir()

			fixedNow := time.Date(2026, 1, 23, 12, 0, 0, 0, time.UTC)

			cache, err := NewAnalysisCache(tmpDir, WithAnalysisCacheNowFunc(func() time.Time { return fixedNow }))
			if err != nil {
				t.Logf("Failed to create analysis cache: %v", err)
				return false
			}

			schema := &PackageConfig{
				URL:    url,
				Parser: "json",
				Path:   "version",
			}

			// Set entry
			cache.Entries[pkg] = AnalysisCacheEntry{
				Schema:    schema,
				Timestamp: fixedNow,
				URL:       url,
			}

			// Call GetWithBypass with bypass=true
			cache.GetWithBypass(pkg, true)

			// Entry should still exist in cache
			entry, exists := cache.GetEntry(pkg)
			if !exists {
				t.Log("Entry should still exist after bypass read")
				return false
			}
			return entry.Schema.URL == url
		},
		genPackageName(),
		genValidURL(),
	))

	// Property: Bypass works for both existing and non-existing entries
	properties.Property("Bypass returns miss for both existing and non-existing entries", prop.ForAll(
		func(existingPkg, nonExistingPkg, url string) bool {
			// Ensure packages are different
			if existingPkg == nonExistingPkg {
				nonExistingPkg = nonExistingPkg + "-other"
			}

			tmpDir := t.TempDir()

			fixedNow := time.Date(2026, 1, 23, 12, 0, 0, 0, time.UTC)

			cache, err := NewAnalysisCache(tmpDir, WithAnalysisCacheNowFunc(func() time.Time { return fixedNow }))
			if err != nil {
				t.Logf("Failed to create analysis cache: %v", err)
				return false
			}

			schema := &PackageConfig{
				URL:    url,
				Parser: "json",
				Path:   "version",
			}

			// Only set entry for existingPkg
			cache.Entries[existingPkg] = AnalysisCacheEntry{
				Schema:    schema,
				Timestamp: fixedNow,
				URL:       url,
			}

			// Both should return miss with bypass=true
			_, foundExisting := cache.GetWithBypass(existingPkg, true)
			_, foundNonExisting := cache.GetWithBypass(nonExistingPkg, true)

			if foundExisting {
				t.Log("Expected miss for existing entry with bypass=true")
				return false
			}
			if foundNonExisting {
				t.Log("Expected miss for non-existing entry with bypass=true")
				return false
			}
			return true
		},
		genPackageName(),
		genPackageName(),
		genValidURL(),
	))

	properties.TestingRun(t)
}
