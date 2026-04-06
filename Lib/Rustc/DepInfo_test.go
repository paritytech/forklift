package Rustc_test

import (
	"forklift/Lib/Rustc"
	"os"
	"path/filepath"
	"testing"
)

// --- ParseDepFile ---

func TestParseDepFile_Simple(t *testing.T) {
	dir := t.TempDir()
	depFile := filepath.Join(dir, "test.d")

	content := "target/release/deps/libfoo-abc123.rlib: src/lib.rs src/util.rs\n"
	os.WriteFile(depFile, []byte(content), 0644)

	deps, err := Rustc.ParseDepFile(depFile)
	if err != nil {
		t.Fatalf("ParseDepFile failed: %s", err)
	}

	if len(deps) != 2 {
		t.Fatalf("Expected 2 deps, got %d", len(deps))
	}
	if deps[0] != "src/lib.rs" || deps[1] != "src/util.rs" {
		t.Fatalf("Unexpected deps: %v", deps)
	}
}

func TestParseDepFile_Continuation(t *testing.T) {
	dir := t.TempDir()
	depFile := filepath.Join(dir, "test.d")

	content := "target/out.rlib: src/a.rs \\\n src/b.rs \\\n src/c.rs\n"
	os.WriteFile(depFile, []byte(content), 0644)

	deps, err := Rustc.ParseDepFile(depFile)
	if err != nil {
		t.Fatalf("ParseDepFile failed: %s", err)
	}

	if len(deps) != 3 {
		t.Fatalf("Expected 3 deps, got %d: %v", len(deps), deps)
	}
}

func TestParseDepFile_RealisticRustc(t *testing.T) {
	dir := t.TempDir()
	depFile := filepath.Join(dir, "libmycrate-abc123.d")

	// Realistic rustc dep-info with absolute paths and build-script outputs
	content := "/work/target/release/deps/libmycrate-abc123.rlib: /work/src/lib.rs \\\n" +
		" /work/src/utils.rs \\\n" +
		" /work/target/release/build/mycrate-def456/out/generated.rs \\\n" +
		" /work/target/release/build/mycrate-def456/out/wasm_binary.rs\n"
	os.WriteFile(depFile, []byte(content), 0644)

	deps, err := Rustc.ParseDepFile(depFile)
	if err != nil {
		t.Fatalf("ParseDepFile failed: %s", err)
	}

	if len(deps) != 4 {
		t.Fatalf("Expected 4 deps, got %d: %v", len(deps), deps)
	}
}

func TestParseDepFile_NoDeps(t *testing.T) {
	dir := t.TempDir()
	depFile := filepath.Join(dir, "test.d")

	content := "target/out.rlib:\n"
	os.WriteFile(depFile, []byte(content), 0644)

	deps, err := Rustc.ParseDepFile(depFile)
	if err != nil {
		t.Fatalf("ParseDepFile failed: %s", err)
	}

	if deps != nil {
		t.Fatalf("Expected nil deps for empty dep list, got %v", deps)
	}
}

func TestParseDepFile_InvalidFormat(t *testing.T) {
	dir := t.TempDir()
	depFile := filepath.Join(dir, "test.d")

	content := "no colon here"
	os.WriteFile(depFile, []byte(content), 0644)

	_, err := Rustc.ParseDepFile(depFile)
	if err == nil {
		t.Fatal("Expected error for invalid format")
	}
}

func TestParseDepFile_NotFound(t *testing.T) {
	_, err := Rustc.ParseDepFile("/nonexistent/path/test.d")
	if err == nil {
		t.Fatal("Expected error for missing file")
	}
}

// --- FilterBuildScriptOutputs ---

func TestFilterBuildScriptOutputs_Mixed(t *testing.T) {
	paths := []string{
		"/work/src/lib.rs",
		"/work/target/release/build/my-crate-abc123/out/generated.rs",
		"/work/target/release/build/my-crate-abc123/out/blob.bin",
		"/work/target/release/deps/libfoo.rlib",
	}

	result := Rustc.FilterBuildScriptOutputs("/work", paths)

	if len(result) != 2 {
		t.Fatalf("Expected 2 build-script outputs, got %d: %v", len(result), result)
	}
}

func TestFilterBuildScriptOutputs_NoneMatch(t *testing.T) {
	paths := []string{
		"/work/src/lib.rs",
		"/work/src/main.rs",
		"/work/target/release/deps/libfoo.rlib",
	}

	result := Rustc.FilterBuildScriptOutputs("/work", paths)

	if len(result) != 0 {
		t.Fatalf("Expected 0 build-script outputs, got %d: %v", len(result), result)
	}
}

func TestFilterBuildScriptOutputs_RelativePaths(t *testing.T) {
	paths := []string{
		"target/release/build/my-crate-abc123/out/generated.rs",
		"src/lib.rs",
	}

	result := Rustc.FilterBuildScriptOutputs("/work", paths)

	if len(result) != 1 {
		t.Fatalf("Expected 1 build-script output, got %d: %v", len(result), result)
	}
}

func TestFilterBuildScriptOutputs_DebugProfile(t *testing.T) {
	paths := []string{
		"/work/target/debug/build/some-crate-789abc/out/file.bin",
	}

	result := Rustc.FilterBuildScriptOutputs("/work", paths)

	if len(result) != 1 {
		t.Fatalf("Expected 1 build-script output for debug profile, got %d", len(result))
	}
}

func TestFilterBuildScriptOutputs_NestedOutDir(t *testing.T) {
	paths := []string{
		"/work/target/release/build/crate-abc/out/subdir/deep/file.rs",
	}

	result := Rustc.FilterBuildScriptOutputs("/work", paths)

	if len(result) != 1 {
		t.Fatalf("Expected 1 build-script output for nested path, got %d", len(result))
	}
}

func TestFilterBuildScriptOutputs_Empty(t *testing.T) {
	result := Rustc.FilterBuildScriptOutputs("/work", []string{})
	if len(result) != 0 {
		t.Fatalf("Expected 0 for empty input, got %d", len(result))
	}
}

// --- ComputeDepInfo ---

func TestComputeDepInfo_Basic(t *testing.T) {
	dir := t.TempDir()

	file1 := filepath.Join(dir, "a.bin")
	file2 := filepath.Join(dir, "b.bin")
	os.WriteFile(file1, []byte("content a"), 0644)
	os.WriteFile(file2, []byte("content b"), 0644)

	record := Rustc.ComputeDepInfo(dir, []string{file1, file2})

	if len(record.Files) != 2 {
		t.Fatalf("Expected 2 files, got %d", len(record.Files))
	}

	// Paths should be relative
	for path := range record.Files {
		if filepath.IsAbs(path) {
			t.Fatalf("Expected relative path, got absolute: %s", path)
		}
	}
}

func TestComputeDepInfo_SkipsMissingFiles(t *testing.T) {
	dir := t.TempDir()

	existing := filepath.Join(dir, "exists.bin")
	os.WriteFile(existing, []byte("data"), 0644)

	record := Rustc.ComputeDepInfo(dir, []string{
		existing,
		filepath.Join(dir, "missing.bin"),
	})

	if len(record.Files) != 1 {
		t.Fatalf("Expected 1 file (skip missing), got %d", len(record.Files))
	}
}

func TestComputeDepInfo_DifferentContentDifferentHash(t *testing.T) {
	dir := t.TempDir()

	file := filepath.Join(dir, "file.bin")

	os.WriteFile(file, []byte("version 1"), 0644)
	record1 := Rustc.ComputeDepInfo(dir, []string{file})

	os.WriteFile(file, []byte("version 2"), 0644)
	record2 := Rustc.ComputeDepInfo(dir, []string{file})

	hash1 := record1.Files["file.bin"]
	hash2 := record2.Files["file.bin"]

	if hash1 == hash2 {
		t.Fatal("Different content should produce different hashes")
	}
}

func TestComputeDepInfo_SameContentSameHash(t *testing.T) {
	dir := t.TempDir()

	file := filepath.Join(dir, "file.bin")

	os.WriteFile(file, []byte("same content"), 0644)
	record1 := Rustc.ComputeDepInfo(dir, []string{file})

	os.WriteFile(file, []byte("same content"), 0644)
	record2 := Rustc.ComputeDepInfo(dir, []string{file})

	hash1 := record1.Files["file.bin"]
	hash2 := record2.Files["file.bin"]

	if hash1 != hash2 {
		t.Fatal("Same content should produce same hashes")
	}
}

// --- Verify ---

func TestVerify_Unchanged(t *testing.T) {
	dir := t.TempDir()

	testFile := filepath.Join(dir, "generated.bin")
	os.WriteFile(testFile, []byte("original content"), 0644)

	record := Rustc.ComputeDepInfo(dir, []string{testFile})

	if !record.Verify(dir) {
		t.Fatal("Verify should pass for unchanged file")
	}
}

func TestVerify_ContentChanged(t *testing.T) {
	dir := t.TempDir()

	testFile := filepath.Join(dir, "generated.bin")
	os.WriteFile(testFile, []byte("original content"), 0644)

	record := Rustc.ComputeDepInfo(dir, []string{testFile})

	os.WriteFile(testFile, []byte("modified content"), 0644)

	if record.Verify(dir) {
		t.Fatal("Verify should fail for modified file")
	}
}

func TestVerify_FileDeleted(t *testing.T) {
	dir := t.TempDir()

	testFile := filepath.Join(dir, "generated.bin")
	os.WriteFile(testFile, []byte("content"), 0644)

	record := Rustc.ComputeDepInfo(dir, []string{testFile})

	os.Remove(testFile)

	if record.Verify(dir) {
		t.Fatal("Verify should fail for missing file")
	}
}

func TestVerify_EmptyRecord(t *testing.T) {
	record := &Rustc.DepInfoRecord{
		Files: map[string]string{},
	}

	if !record.Verify("/tmp") {
		t.Fatal("Empty record should always verify as valid")
	}
}

func TestVerify_MultipleFiles_OneChanged(t *testing.T) {
	dir := t.TempDir()

	file1 := filepath.Join(dir, "a.bin")
	file2 := filepath.Join(dir, "b.bin")
	os.WriteFile(file1, []byte("content a"), 0644)
	os.WriteFile(file2, []byte("content b"), 0644)

	record := Rustc.ComputeDepInfo(dir, []string{file1, file2})

	// Only change one file
	os.WriteFile(file2, []byte("content b modified"), 0644)

	if record.Verify(dir) {
		t.Fatal("Verify should fail when any file changed")
	}
}

// --- Serialize/Deserialize ---

func TestSerializeRoundtrip(t *testing.T) {
	record := &Rustc.DepInfoRecord{
		Files: map[string]string{
			"target/release/build/crate-abc/out/gen.rs":  "aabbccdd",
			"target/release/build/crate-abc/out/blob.bin": "11223344",
		},
	}

	data, err := record.Serialize()
	if err != nil {
		t.Fatalf("Serialize failed: %s", err)
	}

	restored, err := Rustc.DeserializeDepInfo(data)
	if err != nil {
		t.Fatalf("Deserialize failed: %s", err)
	}

	if len(restored.Files) != 2 {
		t.Fatalf("Expected 2 files after roundtrip, got %d", len(restored.Files))
	}

	for path, hash := range record.Files {
		if restored.Files[path] != hash {
			t.Fatalf("Hash mismatch for %s: expected %s, got %s", path, hash, restored.Files[path])
		}
	}
}

func TestDeserialize_Invalid(t *testing.T) {
	_, err := Rustc.DeserializeDepInfo([]byte("not json"))
	if err == nil {
		t.Fatal("Expected error for invalid JSON")
	}
}

// --- FindDepFile ---

func TestFindDepFile_Found(t *testing.T) {
	dir := t.TempDir()
	outDir := filepath.Join(dir, "target", "release", "deps")
	os.MkdirAll(outDir, 0755)

	depFile := filepath.Join(outDir, "mycrate-abc123.d")
	os.WriteFile(depFile, []byte("target: src/lib.rs\n"), 0644)

	result := Rustc.FindDepFile(dir, "target/release/deps", "mycrate")

	if result != depFile {
		t.Fatalf("Expected %s, got %s", depFile, result)
	}
}

func TestFindDepFile_NotFound(t *testing.T) {
	dir := t.TempDir()
	outDir := filepath.Join(dir, "target", "release", "deps")
	os.MkdirAll(outDir, 0755)

	result := Rustc.FindDepFile(dir, "target/release/deps", "nonexistent")

	if result != "" {
		t.Fatalf("Expected empty string for missing dep file, got %s", result)
	}
}

func TestFindDepFile_MultipleMatches_ReturnsNewest(t *testing.T) {
	dir := t.TempDir()
	outDir := filepath.Join(dir, "target", "release", "deps")
	os.MkdirAll(outDir, 0755)

	old := filepath.Join(outDir, "mycrate-old111.d")
	os.WriteFile(old, []byte("target: old.rs\n"), 0644)

	new := filepath.Join(outDir, "mycrate-new222.d")
	os.WriteFile(new, []byte("target: new.rs\n"), 0644)

	result := Rustc.FindDepFile(dir, "target/release/deps", "mycrate")

	// Should return the newest (last written)
	if result != new {
		t.Fatalf("Expected newest dep file %s, got %s", new, result)
	}
}

func TestFindDepFile_AbsoluteOutDir(t *testing.T) {
	dir := t.TempDir()
	outDir := filepath.Join(dir, "target", "release", "deps")
	os.MkdirAll(outDir, 0755)

	depFile := filepath.Join(outDir, "mycrate-abc123.d")
	os.WriteFile(depFile, []byte("target: src/lib.rs\n"), 0644)

	// Pass absolute outDir
	result := Rustc.FindDepFile(dir, outDir, "mycrate")

	if result != depFile {
		t.Fatalf("Expected %s, got %s", depFile, result)
	}
}

// --- End-to-end: issue #30 scenario ---

func TestEndToEnd_StaleCache_BuildScriptOutputChanged(t *testing.T) {
	// Simulates the issue #30 scenario:
	// 1. build.rs generates a file (e.g., compressed WASM blob)
	// 2. Forklift records dep-info with file hash
	// 3. build.rs regenerates the file with different content
	// 4. Dep-info verification catches the change

	dir := t.TempDir()

	// Simulate OUT_DIR structure
	outDirPath := filepath.Join(dir, "target", "release", "build", "my-crate-abc123", "out")
	os.MkdirAll(outDirPath, 0755)

	wasmBlob := filepath.Join(outDirPath, "wasm_binary.rs")

	// Step 1: First build — uncompressed WASM (3 MiB)
	os.WriteFile(wasmBlob, []byte("uncompressed wasm blob content - very large"), 0644)

	// Step 2: Compute and serialize dep-info (as the wrapper would after rustc)
	record := Rustc.ComputeDepInfo(dir, []string{wasmBlob})
	data, err := record.Serialize()
	if err != nil {
		t.Fatalf("Serialize failed: %s", err)
	}

	// Verify cache is valid at this point
	restored, _ := Rustc.DeserializeDepInfo(data)
	if !restored.Verify(dir) {
		t.Fatal("Cache should be valid right after computation")
	}

	// Step 3: build.rs regenerates the file with compressed content
	os.WriteFile(wasmBlob, []byte("compressed wasm - smaller"), 0644)

	// Step 4: Verify catches the change — cache is stale
	if restored.Verify(dir) {
		t.Fatal("Cache should be stale after build-script output changed (issue #30)")
	}
}

func TestEndToEnd_ValidCache_BuildScriptOutputUnchanged(t *testing.T) {
	dir := t.TempDir()

	outDirPath := filepath.Join(dir, "target", "release", "build", "my-crate-abc123", "out")
	os.MkdirAll(outDirPath, 0755)

	genFile := filepath.Join(outDirPath, "generated.rs")
	os.WriteFile(genFile, []byte("// generated code v1"), 0644)

	record := Rustc.ComputeDepInfo(dir, []string{genFile})
	data, _ := record.Serialize()

	// Same build.rs output — rewrite with identical content
	os.WriteFile(genFile, []byte("// generated code v1"), 0644)

	restored, _ := Rustc.DeserializeDepInfo(data)
	if !restored.Verify(dir) {
		t.Fatal("Cache should remain valid when build-script output is unchanged")
	}
}

func TestEndToEnd_ParseFilterComputeVerify(t *testing.T) {
	// Full pipeline: parse .d file → filter → compute → serialize → deserialize → verify

	dir := t.TempDir()

	// Create dep file
	depsDir := filepath.Join(dir, "target", "release", "deps")
	os.MkdirAll(depsDir, 0755)

	// Create build-script output files
	outDir := filepath.Join(dir, "target", "release", "build", "mycrate-hash123", "out")
	os.MkdirAll(outDir, 0755)
	genFile := filepath.Join(outDir, "bindings.rs")
	os.WriteFile(genFile, []byte("generated bindings"), 0644)

	// Create source file (should be filtered out)
	srcDir := filepath.Join(dir, "src")
	os.MkdirAll(srcDir, 0755)
	srcFile := filepath.Join(srcDir, "lib.rs")
	os.WriteFile(srcFile, []byte("fn main() {}"), 0644)

	// Write .d file referencing both
	depContent := depsDir + "/libmycrate-abc.rlib: " + srcFile + " " + genFile + "\n"
	depFile := filepath.Join(depsDir, "mycrate-abc.d")
	os.WriteFile(depFile, []byte(depContent), 0644)

	// Parse
	allDeps, err := Rustc.ParseDepFile(depFile)
	if err != nil {
		t.Fatalf("ParseDepFile: %s", err)
	}
	if len(allDeps) != 2 {
		t.Fatalf("Expected 2 deps, got %d", len(allDeps))
	}

	// Filter — only build-script output
	buildOutputs := Rustc.FilterBuildScriptOutputs(dir, allDeps)
	if len(buildOutputs) != 1 {
		t.Fatalf("Expected 1 build-script output, got %d: %v", len(buildOutputs), buildOutputs)
	}

	// Compute
	record := Rustc.ComputeDepInfo(dir, buildOutputs)
	if len(record.Files) != 1 {
		t.Fatalf("Expected 1 file in record, got %d", len(record.Files))
	}

	// Roundtrip
	data, _ := record.Serialize()
	restored, _ := Rustc.DeserializeDepInfo(data)

	// Verify — unchanged
	if !restored.Verify(dir) {
		t.Fatal("Should verify OK when unchanged")
	}

	// Modify build-script output
	os.WriteFile(genFile, []byte("regenerated bindings v2"), 0644)

	// Verify — changed
	if restored.Verify(dir) {
		t.Fatal("Should detect changed build-script output")
	}
}
