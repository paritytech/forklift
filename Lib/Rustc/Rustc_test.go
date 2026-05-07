package Rustc_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"forklift/FileManager/Models"
	"forklift/Lib/Config"
	"forklift/Lib/Logging"
	"forklift/Lib/Rustc"
	"os"
	"path"
	"path/filepath"
	"reflect"
	"testing"
)

func TestCmdTool_GetExternDeps(t *testing.T) {

	var input = []string{"qwerty", "asdfgh", "--extern", "a=a/b/c", "--extern", "d=d/e/f", "--extern", "g=g/h/i", "-extern", "j=j/k/l"}

	var nonBasePathResult = Rustc.GetExternDeps(&input, false)
	if !reflect.DeepEqual(*nonBasePathResult, []string{"a/b/c", "d/e/f", "g/h/i"}) {
		t.Error("Test failed")
	}

	var onlyBasePathResult = Rustc.GetExternDeps(&input, true)
	if !reflect.DeepEqual(*onlyBasePathResult, []string{"c", "f", "i"}) {
		t.Error("Test failed")
	}
}

func TestWrapperTool_WriteStderrFile(t *testing.T) {
	var wd, _ = os.Getwd()
	var wrapper = Rustc.NewWrapperToolFromArgs(wd, []string{"aaaa", "bbbb"})

	var data = "{\"artifact\":\"deps/base64-a62ed92405ecbfa1.d\",\"emit\":\"dep-info\"}"
	var expectedData = "{\"artifact\":\"base64-a62ed92405ecbfa1.d\",\"emit\":\"dep-info\"}\n"

	var reader = bytes.NewReader([]byte(data))

	wrapper.Logger = Logging.CreateLogger("wrapper", 2, nil)
	var artifacts = wrapper.WriteStderrFile(reader)

	if len(*artifacts) != 1 {
		t.Error("No artifact")
	}

	if (*artifacts)[0].Artifact != "base64-a62ed92405ecbfa1.d" {
		t.Error("Wrong artifact")
	}

	var fileData, _ = os.ReadFile("target/forklift/" + wrapper.GetCachePackageName() + "-stderr")

	var actualData = string(fileData)
	if actualData != expectedData {
		t.Error("Data mismatch")
	}

}

func TestWrapperTool_ReadStderrFile(t *testing.T) {
	var wd, _ = os.Getwd()
	var wrapper = Rustc.NewWrapperToolFromArgs(wd, []string{"-a", "b"})

	var dataBytes, _ = json.Marshal(Rustc.Artifact{
		Artifact: filepath.Join("target", "deps", "base64-a62ed92405ecbfa1.d"),
	})
	var data = string(dataBytes)

	var itemsCachePath = path.Join(wd, "target", "forklift")
	os.MkdirAll(itemsCachePath, 0755)
	os.WriteFile("target/forklift/"+wrapper.GetCachePackageName()+"-stderr", []byte(data), 0755)

	var expectedBytes, _ = json.Marshal(Rustc.Artifact{
		Artifact: filepath.Join(wd, "target", "deps", "base64-a62ed92405ecbfa1.d"),
	})
	var expectedData = string(expectedBytes) + "\n"

	var reader = wrapper.ReadStderrFile()
	var buf = bytes.Buffer{}
	buf.ReadFrom(reader)

	var actualData = buf.String()
	if actualData != expectedData {
		fmt.Printf("Expected: %s\n", expectedData)
		fmt.Printf("Actual  : %s\n", actualData)
		t.Error("Data mismatch")
	}
}

// TestNewWrapperToolFromCacheItem_UsesStoredCachePackageName is the direct
// regression test for the uploader/wrapper name-mismatch bug. The uploader
// MUST return the name shipped in the CacheItem rather than recomputing one
// — if it recomputes, the result is a sha of mostly-empty inputs which won't
// match the precomputed storedName used to build the on-disk *-stderr path.
func TestNewWrapperToolFromCacheItem_UsesStoredCachePackageName(t *testing.T) {
	const storedName = "some_crate_deadbeef00000000000000000000000000cafe"
	item := Models.CacheItem{
		Name:             "some_crate",
		CachePackageName: storedName,
		// Fields below only exist to keep GetCachePackageName from panicking
		// if a regression makes it recompute (ExternDepsChecksum needs a
		// non-empty CrateExternDepsChecksum to skip dereferencing rustcArgs).
		// If recomputation ever kicks in, the sha of these inputs will not
		// match storedName and we get a clean assertion failure.
		CrateExternDepsChecksum: "0",
	}
	wrapper := Rustc.NewWrapperToolFromCacheItem(t.TempDir(), item)
	if got := wrapper.GetCachePackageName(); got != storedName {
		t.Errorf("expected stored name %q, got recomputed %q", storedName, got)
	}
}

// TestCachePackageName_SurvivesExtraEnvVarDrift reproduces the production
// scenario: the wrapper subprocess sees a Cache.ExtraEnv var (e.g. a cargo-
// injected CARGO_* / OUT_DIR value) that the parent server process does not.
// The on-disk path the wrapper wrote and the path the uploader looks up must
// still agree after the roundtrip through CacheItem.
func TestCachePackageName_SurvivesExtraEnvVarDrift(t *testing.T) {
	const varName = "FORKLIFT_TEST_EXTRA_VAR"

	t.Setenv(varName, "wrapper-value")
	origExtraEnv := Config.AppConfig.Cache.ExtraEnv
	Config.AppConfig.Cache.ExtraEnv = []string{varName}
	t.Cleanup(func() { Config.AppConfig.Cache.ExtraEnv = origExtraEnv })

	wd, _ := os.Getwd()

	// Wrapper side: var is set, compute and serialize.
	wrapper := Rustc.NewWrapperToolFromArgs(wd, []string{"aaaa", "bbbb"})
	wrapperName := wrapper.GetCachePackageName()
	item := wrapper.ToCacheItem()

	// Server side: var is absent (simulates the parent process env).
	os.Unsetenv(varName)

	uploader := Rustc.NewWrapperToolFromCacheItem(wd, item)
	uploaderName := uploader.GetCachePackageName()

	if wrapperName != uploaderName {
		t.Errorf("cache-package name drifted across env change\n  wrapper:  %s\n  uploader: %s", wrapperName, uploaderName)
	}
}

// TestExtraEnvVarsChecksum_EnvVarChangesWrapperName locks in the intended
// behavior on the wrapper side: changing a value listed in Cache.ExtraEnv
// actually changes the cache-package name. Without this check, a future
// refactor could silently stop honoring ExtraEnv and the
// SurvivesExtraEnvVarDrift test above would pass trivially.
func TestExtraEnvVarsChecksum_EnvVarChangesWrapperName(t *testing.T) {
	const varName = "FORKLIFT_TEST_SENSITIVE_VAR"

	origExtraEnv := Config.AppConfig.Cache.ExtraEnv
	Config.AppConfig.Cache.ExtraEnv = []string{varName}
	t.Cleanup(func() { Config.AppConfig.Cache.ExtraEnv = origExtraEnv })

	wd, _ := os.Getwd()

	t.Setenv(varName, "value-one")
	nameOne := Rustc.NewWrapperToolFromArgs(wd, []string{"aaaa", "bbbb"}).GetCachePackageName()

	t.Setenv(varName, "value-two")
	nameTwo := Rustc.NewWrapperToolFromArgs(wd, []string{"aaaa", "bbbb"}).GetCachePackageName()

	if nameOne == nameTwo {
		t.Errorf("expected different names for different values of %s, both were %q", varName, nameOne)
	}
}

// TestReadStderrFile_MissingFileReturnsEmpty covers the secondary fix: when
// the stderr file is absent, ReadStderrFile must surface the error via the
// logger and return an empty reader — never a nil *os.File wrapped in an
// interface, which previously caused Scanner to silently produce zero lines.
func TestReadStderrFile_MissingFileReturnsEmpty(t *testing.T) {
	wd := t.TempDir()

	item := Models.CacheItem{
		Name:                    "nonexistent",
		CachePackageName:        "nonexistent_missingstderrfile",
		CrateExternDepsChecksum: "0",
	}
	wrapper := Rustc.NewWrapperToolFromCacheItem(wd, item)
	wrapper.Logger = Logging.CreateLogger("test", 2, nil)

	reader := wrapper.ReadStderrFile()
	buf := bytes.Buffer{}
	n, err := buf.ReadFrom(reader)
	if err != nil {
		t.Fatalf("unexpected read error: %s", err)
	}
	if n != 0 {
		t.Errorf("expected empty buffer, got %d bytes", n)
	}
}
