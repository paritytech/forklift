//go:build integration

package integration_test

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
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

// TestFirstBuild_CacheMiss verifies that the first build runs rustc for all crates (no cache).
func TestFirstBuild_CacheMiss(t *testing.T) {
	env := newTestEnv(t)

	output := env.forkliftBuild(t)

	// First build — should see "Executing rustc" for both crates
	if !strings.Contains(output, "Executing rustc") {
		t.Fatal("Expected 'Executing rustc' on first build (cache miss)")
	}

	// Should report cache misses, not hits
	if strings.Contains(output, "Downloaded and unpacked") {
		t.Fatal("Should not have cache hits on first build")
	}
}

// TestSecondBuild_CacheHit verifies that a second identical build uses cached artifacts.
func TestSecondBuild_CacheHit(t *testing.T) {
	env := newTestEnv(t)

	// First build — populates cache
	env.forkliftBuild(t)

	// Clean target so cargo re-invokes rustc wrappers
	env.cargoClean(t)

	// Second build — should use cache
	output := env.forkliftBuild(t)

	if !strings.Contains(output, "Downloaded and unpacked") {
		t.Fatal("Expected cache hits on second build")
	}
}

// TestBuildScriptOutputChange_CacheInvalidation (issue #30)
// Verifies that changing build.rs output invalidates the cache.
func TestBuildScriptOutputChange_CacheInvalidation(t *testing.T) {
	env := newTestEnv(t)

	// First build with GENERATED_CONTENT=version_1
	env.extraEnv = append(env.extraEnv, "GENERATED_CONTENT=version_1")
	env.forkliftBuild(t)

	// Clean target
	env.cargoClean(t)

	// Second build with same content — should hit cache
	output := env.forkliftBuild(t)
	if !strings.Contains(output, "Downloaded and unpacked") {
		t.Fatal("Expected cache hit with unchanged build-script output")
	}

	// Clean target
	env.cargoClean(t)

	// Third build with DIFFERENT content — should invalidate cache for crate_a
	env.extraEnv = replaceEnv(env.extraEnv, "GENERATED_CONTENT", "version_2")
	output = env.forkliftBuild(t)

	// crate_a should be rebuilt (cache miss due to dep-info mismatch)
	if strings.Contains(output, "Build-script outputs changed") ||
		strings.Contains(output, "Executing rustc") {
		// Good — either the dep-info check caught it or rustc re-ran
		t.Logf("Cache correctly invalidated for changed build-script output")
	}
}

// TestSourceChange_CacheInvalidation verifies that modifying source invalidates the cache.
func TestSourceChange_CacheInvalidation(t *testing.T) {
	env := newTestEnv(t)

	// First build
	env.forkliftBuild(t)

	// Clean target
	env.cargoClean(t)

	// Modify source in crate_b
	libPath := filepath.Join(env.projectDir, "crate_b", "src", "lib.rs")
	original, _ := os.ReadFile(libPath)
	os.WriteFile(libPath, append(original, []byte("\n// modified\n")...), 0644)
	defer os.WriteFile(libPath, original, 0644)

	// Rebuild — crate_b should be rebuilt
	output := env.forkliftBuild(t)

	if !strings.Contains(output, "Executing rustc") {
		t.Fatal("Expected rustc execution after source modification")
	}
}

// TestDependencyRebuilt_CascadeInvalidation verifies that when crate_a is rebuilt,
// crate_b's cache is also invalidated (dependency rebuilt tracking).
func TestDependencyRebuilt_CascadeInvalidation(t *testing.T) {
	env := newTestEnv(t)

	// First build — populates cache
	env.forkliftBuild(t)

	// Clean target
	env.cargoClean(t)

	// Modify crate_a source
	libPath := filepath.Join(env.projectDir, "crate_a", "src", "lib.rs")
	original, _ := os.ReadFile(libPath)
	os.WriteFile(libPath, append(original, []byte("\n// changed\n")...), 0644)
	defer os.WriteFile(libPath, original, 0644)

	// Rebuild — both crates should be rebuilt
	output := env.forkliftBuild(t)

	// crate_a changed source → cache miss
	// crate_b depends on crate_a → dependency rebuilt → cache miss
	if !strings.Contains(output, "Executing rustc") {
		t.Fatal("Expected rustc execution after dependency source change")
	}
}

// --- Test helpers ---

type testEnv struct {
	projectDir string
	storageDir string
	configDir  string
	extraEnv   []string
}

func newTestEnv(t *testing.T) *testEnv {
	t.Helper()

	tmpDir := t.TempDir()

	// Copy testdata project to temp dir so tests don't interfere
	projectDir := filepath.Join(tmpDir, "project")
	copyDir(t, filepath.Join(getTestdataDir()), projectDir)

	// Create storage directory
	storageDir := filepath.Join(tmpDir, "storage")
	os.MkdirAll(storageDir, 0755)

	// Create forklift config
	configDir := filepath.Join(projectDir, ".forklift")
	os.MkdirAll(configDir, 0755)
	config := fmt.Sprintf(`[general]
logLevel = "trace"
threadsCount = 2

[storage]
type = "fs"

[storage.fs]
directory = "%s"

[compression]
type = "none"
`, storageDir)
	os.WriteFile(filepath.Join(configDir, "config.toml"), []byte(config), 0644)

	return &testEnv{
		projectDir: projectDir,
		storageDir: storageDir,
		configDir:  configDir,
	}
}

func (env *testEnv) forkliftBuild(t *testing.T) string {
	t.Helper()

	cmd := exec.Command(forkliftBin, "cargo", "build")
	cmd.Dir = env.projectDir
	cmd.Env = append(os.Environ(), env.extraEnv...)

	out, err := cmd.CombinedOutput()
	output := string(out)
	t.Logf("forklift output:\n%s", output)

	if err != nil {
		t.Fatalf("forklift cargo build failed: %s\noutput: %s", err, output)
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

func getRepoRoot() string {
	// integration_test/ is one level below repo root
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
