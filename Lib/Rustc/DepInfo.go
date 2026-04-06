package Rustc

import (
	"crypto/sha1"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// DepInfoRecord stores file paths and their hashes from rustc's dep-info output.
// Used to detect stale cache when build.rs-generated files change (e.g. include_bytes! targets).
type DepInfoRecord struct {
	Files map[string]string `json:"files"` // relative path → sha1 hex
}

// ParseDepFile parses a rustc .d (Makefile-format) dep-info file and returns the list of dependency file paths.
// Format: "target: dep1 dep2 dep3" with backslash line continuations.
func ParseDepFile(depFilePath string) ([]string, error) {
	data, err := os.ReadFile(depFilePath)
	if err != nil {
		return nil, err
	}

	content := string(data)

	// Remove backslash-newline continuations
	content = strings.ReplaceAll(content, "\\\n", " ")

	// Split on colon to get deps part (everything after first ":")
	parts := strings.SplitN(content, ":", 2)
	if len(parts) < 2 {
		return nil, fmt.Errorf("invalid dep-info format in %s", depFilePath)
	}

	depsStr := strings.TrimSpace(parts[1])
	if depsStr == "" {
		return nil, nil
	}

	fields := strings.Fields(depsStr)
	return fields, nil
}

// FilterBuildScriptOutputs returns only paths that are under a cargo build script output directory
// (matching target/*/build/*/out/ pattern), which are the files not tracked by forklift's cache key.
func FilterBuildScriptOutputs(workDir string, paths []string) []string {
	var result []string
	for _, p := range paths {
		rel := p
		if filepath.IsAbs(p) {
			var err error
			rel, err = filepath.Rel(workDir, p)
			if err != nil {
				continue
			}
		}

		// Match paths like target/<profile>/build/<crate>-<hash>/out/...
		parts := strings.Split(filepath.ToSlash(rel), "/")
		if len(parts) >= 5 &&
			parts[0] == "target" &&
			parts[2] == "build" &&
			parts[4] == "out" {
			result = append(result, p)
		}
	}
	return result
}

// ComputeDepInfo hashes each file and returns a DepInfoRecord with paths relative to workDir.
func ComputeDepInfo(workDir string, paths []string) *DepInfoRecord {
	record := &DepInfoRecord{
		Files: make(map[string]string),
	}

	for _, p := range paths {
		hash, err := hashFile(p)
		if err != nil {
			continue
		}

		rel := p
		if filepath.IsAbs(p) {
			r, err := filepath.Rel(workDir, p)
			if err == nil {
				rel = r
			}
		}

		record.Files[rel] = hash
	}

	return record
}

// Verify checks that all files in the record still match their stored hashes.
// Returns true if all files match (cache is still valid), false if any changed.
func (d *DepInfoRecord) Verify(workDir string) bool {
	if len(d.Files) == 0 {
		return true
	}

	for relPath, expectedHash := range d.Files {
		absPath := relPath
		if !filepath.IsAbs(relPath) {
			absPath = filepath.Join(workDir, relPath)
		}

		currentHash, err := hashFile(absPath)
		if err != nil {
			// File doesn't exist or can't be read — stale cache
			return false
		}

		if currentHash != expectedHash {
			return false
		}
	}

	return true
}

func (d *DepInfoRecord) Serialize() ([]byte, error) {
	return json.Marshal(d)
}

func DeserializeDepInfo(data []byte) (*DepInfoRecord, error) {
	var record DepInfoRecord
	err := json.Unmarshal(data, &record)
	if err != nil {
		return nil, err
	}
	return &record, nil
}

// FindDepFile locates the .d file for a crate in the output directory.
func FindDepFile(workDir string, outDir string, crateName string) string {
	dir := outDir
	if !filepath.IsAbs(dir) {
		dir = filepath.Join(workDir, outDir)
	}

	pattern := filepath.Join(dir, crateName+"-*.d")
	matches, err := filepath.Glob(pattern)
	if err != nil || len(matches) == 0 {
		return ""
	}

	// Return the most recently modified match
	var newest string
	var newestTime int64
	for _, m := range matches {
		info, err := os.Stat(m)
		if err != nil {
			continue
		}
		if info.ModTime().UnixNano() > newestTime {
			newestTime = info.ModTime().UnixNano()
			newest = m
		}
	}

	return newest
}

func hashFile(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	h := sha1.New()
	_, err = io.Copy(h, f)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%x", h.Sum(nil)), nil
}
