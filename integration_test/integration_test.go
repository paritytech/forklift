//go:build integration

package integration_test

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

var forkliftBin string

func TestMain(m *testing.M) {
	// Build forklift binary once for all tests
	tmpDir, err := os.MkdirTemp("", "forklift-integration-*")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create temp dir: %s\n", err)
		os.Exit(1)
	}
	defer os.RemoveAll(tmpDir)

	forkliftBin = filepath.Join(tmpDir, "forklift")
	cmd := exec.Command("go", "build", "-o", forkliftBin, ".")
	cmd.Dir = getRepoRoot()
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to build forklift: %s\n", err)
		os.Exit(1)
	}

	os.Exit(m.Run())
}

// --- Core cache behavior ---

// TestFirstBuild_CacheMiss verifies that the first build runs rustc for all crates (no cache).
func TestFirstBuild_CacheMiss(t *testing.T) {
	env := newTestEnv(t, nil)

	output := env.forkliftBuild(t)

	if !strings.Contains(output, "Executing rustc") {
		t.Fatal("Expected 'Executing rustc' on first build (cache miss)")
	}
	if strings.Contains(output, "Downloaded and unpacked") {
		t.Fatal("Should not have cache hits on first build")
	}

	// Verify cache report shows misses
	report := parseCacheReport(output)
	if report.cacheHit != 0 {
		t.Fatalf("Expected 0 cache hits on first build, got %d", report.cacheHit)
	}
	if report.totalCrates == 0 {
		t.Fatal("Expected non-zero total crates")
	}
}

// TestSecondBuild_CacheHit verifies that a second identical build uses cached artifacts.
func TestSecondBuild_CacheHit(t *testing.T) {
	env := newTestEnv(t, nil)

	// First build — populates cache
	env.forkliftBuild(t)
	env.cargoClean(t)

	// Second build — should use cache
	output := env.forkliftBuild(t)

	if !strings.Contains(output, "Downloaded and unpacked") {
		t.Fatal("Expected cache hits on second build")
	}

	report := parseCacheReport(output)
	if report.cacheHit == 0 {
		t.Fatal("Expected non-zero cache hits on second build")
	}
	if report.cacheMiss != 0 {
		t.Fatalf("Expected 0 cache misses on second build, got %d", report.cacheMiss)
	}
}

// TestThirdBuild_CacheStillValid verifies cache persists across multiple clean+build cycles.
func TestThirdBuild_CacheStillValid(t *testing.T) {
	env := newTestEnv(t, nil)

	env.forkliftBuild(t)
	env.cargoClean(t)
	env.forkliftBuild(t)
	env.cargoClean(t)

	// Third build — cache should still be valid
	output := env.forkliftBuild(t)

	report := parseCacheReport(output)
	if report.cacheHit == 0 {
		t.Fatal("Expected cache hits on third build")
	}
}

// --- Source & dependency invalidation ---

// TestSourceChange_CacheInvalidation verifies that modifying source invalidates the cache.
func TestSourceChange_CacheInvalidation(t *testing.T) {
	env := newTestEnv(t, nil)

	env.forkliftBuild(t)
	env.cargoClean(t)

	// Modify source in crate_b
	libPath := filepath.Join(env.projectDir, "crate_b", "src", "lib.rs")
	original, _ := os.ReadFile(libPath)
	os.WriteFile(libPath, append(original, []byte("\n// modified\n")...), 0644)
	defer os.WriteFile(libPath, original, 0644)

	output := env.forkliftBuild(t)

	if !strings.Contains(output, "Executing rustc") {
		t.Fatal("Expected rustc execution after source modification")
	}
}

// TestDependencyRebuilt_CascadeInvalidation verifies that when crate_a is rebuilt,
// crate_b's cache is also invalidated (dependency rebuilt tracking).
func TestDependencyRebuilt_CascadeInvalidation(t *testing.T) {
	env := newTestEnv(t, nil)

	env.forkliftBuild(t)
	env.cargoClean(t)

	// Modify crate_a source
	libPath := filepath.Join(env.projectDir, "crate_a", "src", "lib.rs")
	original, _ := os.ReadFile(libPath)
	os.WriteFile(libPath, append(original, []byte("\n// changed\n")...), 0644)
	defer os.WriteFile(libPath, original, 0644)

	output := env.forkliftBuild(t)

	// crate_a changed source → new cache key → cache miss → rustc runs
	// crate_b depends on crate_a → either "dependency rebuilt" (if crate_b's key matched old cache)
	// or cache miss (if crate_b's key also changed due to extern dep checksum change)
	if !strings.Contains(output, "Executing rustc") {
		t.Fatal("Expected rustc execution after dependency source change")
	}

	report := parseCacheReport(output)
	// Both crate_a and crate_b should be rebuilt (either via miss or dep-rebuilt)
	if report.cacheMiss+report.depRebuilt < 2 {
		t.Fatalf("Expected at least 2 crates rebuilt (miss + dep_rebuilt), got miss=%d dep_rebuilt=%d",
			report.cacheMiss, report.depRebuilt)
	}
}

// --- Build script / dep-info (issue #30) ---

// TestBuildScriptOutputChange_CacheInvalidation verifies that changing build.rs output
// invalidates the cache via dep-info validation.
func TestBuildScriptOutputChange_CacheInvalidation(t *testing.T) {
	env := newTestEnv(t, nil)

	// First build with GENERATED_CONTENT=version_1
	env.extraEnv = append(env.extraEnv, "GENERATED_CONTENT=version_1")
	env.forkliftBuild(t)
	env.cargoClean(t)

	// Second build with same content — should hit cache
	output := env.forkliftBuild(t)
	if !strings.Contains(output, "Downloaded and unpacked") {
		t.Fatal("Expected cache hit with unchanged build-script output")
	}
	env.cargoClean(t)

	// Third build with DIFFERENT content — should invalidate cache for crate_a
	env.extraEnv = replaceEnv(env.extraEnv, "GENERATED_CONTENT", "version_2")
	output = env.forkliftBuild(t)

	if !strings.Contains(output, "Build-script outputs changed") {
		t.Fatal("Expected dep-info invalidation ('Build-script outputs changed') for changed build-script output")
	}
}

// TestBuildScriptOutputChange_CacheUpdated verifies that after invalidation,
// the new cache entry works correctly on subsequent builds.
func TestBuildScriptOutputChange_CacheUpdated(t *testing.T) {
	env := newTestEnv(t, nil)

	// Build with version_1
	env.extraEnv = append(env.extraEnv, "GENERATED_CONTENT=version_1")
	env.forkliftBuild(t)
	env.cargoClean(t)

	// Build with version_2 — invalidates and re-caches
	env.extraEnv = replaceEnv(env.extraEnv, "GENERATED_CONTENT", "version_2")
	env.forkliftBuild(t)
	env.cargoClean(t)

	// Build with version_2 again — should hit the UPDATED cache
	output := env.forkliftBuild(t)

	report := parseCacheReport(output)
	if report.cacheHit == 0 {
		t.Fatal("Expected cache hits after cache was updated with new build-script output")
	}
	if strings.Contains(output, "Build-script outputs changed") {
		t.Fatal("Should NOT invalidate cache when build-script output matches the updated cache")
	}
}

// --- Compression ---

// TestCacheWithZstdCompression verifies caching works with zstd compression enabled.
func TestCacheWithZstdCompression(t *testing.T) {
	env := newTestEnv(t, &testConfig{compression: "zstd"})

	env.forkliftBuild(t)
	env.cargoClean(t)

	output := env.forkliftBuild(t)

	if !strings.Contains(output, "compressor: zstd") {
		t.Fatal("Expected zstd compressor to be active")
	}

	report := parseCacheReport(output)
	if report.cacheHit == 0 {
		t.Fatal("Expected cache hits with zstd compression")
	}
}

// --- Bypass modes ---

// TestForkliftBypass verifies FORKLIFT_BYPASS=true skips forklift entirely.
func TestForkliftBypass(t *testing.T) {
	env := newTestEnv(t, nil)

	env.extraEnv = append(env.extraEnv, "FORKLIFT_BYPASS=true")
	output := env.forkliftBuild(t)

	// Should not see forklift-specific output
	if strings.Contains(output, "Using forklift") {
		t.Fatal("Expected forklift to be bypassed")
	}
	if strings.Contains(output, "Cache report") {
		t.Fatal("Should not see cache report when bypassed")
	}
}

// TestNullStorage verifies that null storage gracefully results in no caching.
func TestNullStorage(t *testing.T) {
	env := newTestEnv(t, &testConfig{storageType: "null"})

	env.forkliftBuild(t)
	env.cargoClean(t)

	output := env.forkliftBuild(t)

	if !strings.Contains(output, "storage: null") {
		t.Fatal("Expected null storage")
	}

	// With null storage, nothing is cached — everything is a miss
	report := parseCacheReport(output)
	if report.cacheHit != 0 {
		t.Fatalf("Expected 0 cache hits with null storage, got %d", report.cacheHit)
	}
}

// --- Quiet mode ---

// TestQuietMode verifies that quiet mode suppresses forklift output while build still succeeds.
func TestQuietMode(t *testing.T) {
	env := newTestEnv(t, &testConfig{quiet: true})

	output := env.forkliftBuild(t)

	// Quiet mode should suppress "Using forklift" banner and cache report
	if strings.Contains(output, "Using forklift") {
		t.Fatal("Expected 'Using forklift' banner to be suppressed in quiet mode")
	}
	if strings.Contains(output, "Cache report") {
		t.Fatal("Expected cache report to be suppressed in quiet mode")
	}
}

// --- Release profile ---

// TestReleaseBuild verifies caching works for release builds.
func TestReleaseBuild(t *testing.T) {
	env := newTestEnv(t, nil)

	env.forkliftBuildRelease(t)
	env.cargoClean(t)

	output := env.forkliftBuildRelease(t)

	report := parseCacheReport(output)
	if report.cacheHit == 0 {
		t.Fatal("Expected cache hits on second release build")
	}
}

// --- Profiles don't cross-contaminate ---

// TestDebugAndReleaseSeparateCache verifies debug and release builds use different cache keys.
func TestDebugAndReleaseSeparateCache(t *testing.T) {
	env := newTestEnv(t, nil)

	// Build debug — populates cache
	env.forkliftBuild(t)
	env.cargoClean(t)

	// Build release — should NOT hit the debug cache
	output := env.forkliftBuildRelease(t)

	report := parseCacheReport(output)
	if report.cacheHit != 0 {
		t.Fatalf("Expected 0 cache hits (release should not reuse debug cache), got %d", report.cacheHit)
	}
}

// --- Storage files ---

// TestCacheFilesCreated verifies that cache files are actually written to storage.
func TestCacheFilesCreated(t *testing.T) {
	env := newTestEnv(t, nil)

	env.forkliftBuild(t)

	// Check that storage directory is not empty
	entries, err := os.ReadDir(env.storageDir)
	if err != nil {
		t.Fatalf("Failed to read storage dir: %s", err)
	}

	if len(entries) == 0 {
		t.Fatal("Expected cache files in storage directory after first build")
	}

	// Should have .meta sidecar files for metadata
	hasMetaFile := false
	for _, entry := range entries {
		if strings.HasSuffix(entry.Name(), ".meta") {
			hasMetaFile = true
			break
		}
	}
	if !hasMetaFile {
		t.Fatal("Expected .meta sidecar files in storage directory")
	}
}

// --- Test helpers ---

type testConfig struct {
	storageType string
	compression string
	quiet       bool
}

type testEnv struct {
	projectDir string
	storageDir string
	configDir  string
	extraEnv   []string
}

func newTestEnv(t *testing.T, cfg *testConfig) *testEnv {
	t.Helper()

	tmpDir := t.TempDir()

	// Copy testdata project to temp dir so tests don't interfere
	projectDir := filepath.Join(tmpDir, "project")
	copyDir(t, getTestdataDir(), projectDir)

	// Create storage directory
	storageDir := filepath.Join(tmpDir, "storage")
	os.MkdirAll(storageDir, 0755)

	// Defaults
	storageType := "fs"
	compression := "none"
	quiet := false
	if cfg != nil {
		if cfg.storageType != "" {
			storageType = cfg.storageType
		}
		if cfg.compression != "" {
			compression = cfg.compression
		}
		quiet = cfg.quiet
	}

	// Create forklift config
	configDir := filepath.Join(projectDir, ".forklift")
	os.MkdirAll(configDir, 0755)
	config := fmt.Sprintf(`[general]
logLevel = "trace"
threadsCount = 2
quiet = %t

[storage]
type = "%s"

[storage.fs]
directory = "%s"

[compression]
type = "%s"

[compression.zstd]
compressionLevel = 3
`, quiet, storageType, storageDir, compression)
	os.WriteFile(filepath.Join(configDir, "config.toml"), []byte(config), 0644)

	return &testEnv{
		projectDir: projectDir,
		storageDir: storageDir,
		configDir:  configDir,
	}
}

func (env *testEnv) forkliftBuild(t *testing.T, extraArgs ...string) string {
	t.Helper()
	args := append([]string{"cargo", "build"}, extraArgs...)
	return env.runForklift(t, args...)
}

func (env *testEnv) forkliftBuildRelease(t *testing.T) string {
	t.Helper()
	return env.runForklift(t, "cargo", "build", "--release")
}

func (env *testEnv) runForklift(t *testing.T, args ...string) string {
	t.Helper()

	cmd := exec.Command(forkliftBin, args...)
	cmd.Dir = env.projectDir
	cmd.Env = append(os.Environ(), env.extraEnv...)

	out, err := cmd.CombinedOutput()
	output := string(out)
	t.Logf("forklift output:\n%s", output)

	if err != nil {
		t.Fatalf("forklift %s failed: %s\noutput: %s", strings.Join(args, " "), err, output)
	}

	return output
}

func (env *testEnv) cargoClean(t *testing.T) {
	t.Helper()

	cmd := exec.Command("cargo", "clean")
	cmd.Dir = env.projectDir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("cargo clean failed: %s\n%s", err, out)
	}
}

// parseCacheReport extracts key numbers from the forklift cache report output.
type cacheReport struct {
	totalCrates int
	cacheHit    int
	cacheMiss   int
	depRebuilt  int
}

func parseCacheReport(output string) cacheReport {
	return cacheReport{
		totalCrates: parseReportField(output, `Total crates processed:\s+(\d+)`),
		cacheHit:    parseReportField(output, `Cache hit:\s+(\d+)`),
		cacheMiss:   parseReportField(output, `Cache miss:\s+(\d+)`),
		depRebuilt:  parseReportField(output, `Dependency rebuilt:\s+(\d+)`),
	}
}

func parseReportField(output, pattern string) int {
	re := regexp.MustCompile(pattern)
	match := re.FindStringSubmatch(output)
	if len(match) < 2 {
		return 0
	}
	val, _ := strconv.Atoi(match[1])
	return val
}

func getRepoRoot() string {
	dir, _ := filepath.Abs(filepath.Join(filepath.Dir("."), ".."))
	return dir
}

func getTestdataDir() string {
	dir, _ := filepath.Abs("testdata")
	return dir
}

func copyDir(t *testing.T, src, dst string) {
	t.Helper()
	cmd := exec.Command("cp", "-r", src, dst)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("Failed to copy %s to %s: %s\n%s", src, dst, err, out)
	}
}

func replaceEnv(envs []string, key, value string) []string {
	prefix := key + "="
	for i, e := range envs {
		if strings.HasPrefix(e, prefix) {
			envs[i] = prefix + value
			return envs
		}
	}
	return append(envs, prefix+value)
}
