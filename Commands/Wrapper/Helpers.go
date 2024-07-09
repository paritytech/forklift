package Wrapper

import (
	"crypto/sha1"
	"fmt"
	log "forklift/Lib/Logging/ConsoleLogger"
	"forklift/Lib/Rustc"
	"hash"
	"io"
	"os"
	"path/filepath"
)

func hasCargoToml(path string) bool {
	var cargoTomls, _ = filepath.Glob(filepath.Join(path, "Cargo.toml"))
	return len(cargoTomls) > 0
}

func calcChecksum(wrapperTool *Rustc.WrapperTool) bool {

	var path = wrapperTool.SourceFile

	path = filepath.Dir(path)

	for {
		if hasCargoToml(path) {
			break
		} else {
			path = filepath.Dir(path)
		}
	}

	wrapperTool.Logger.Tracef("Cargo.toml found in %s", path)

	var sha = sha1.New()
	checksum(path, sha, true)
	wrapperTool.CrateSourceChecksum = fmt.Sprintf("%x", sha.Sum(nil))
	return true
}

func checksum(path string, hash hash.Hash, root bool) {
	var entries, _ = os.ReadDir(path)

	/*if !root && hasCargoToml(path) {
		return
	}*/

	for _, entry := range entries {

		if root && needIgnore(entry.Name()) {
			continue
		}

		if entry.IsDir() {
			checksum(filepath.Join(path, entry.Name()), hash, false)
		} else {
			var file, _ = os.Open(filepath.Join(path, entry.Name()))
			if log.GetLevel() > log.DebugLevel {
				log.Tracef("calculating checksum of %s, result %s", filepath.Join(path, entry.Name()), fmt.Sprintf("%x", hash.Sum(nil)))
			}
			io.Copy(hash, file)
			file.Close()
		}
	}
}

// needIgnore returns true if entryName should be ignored
func needIgnore(entryName string) bool {
	var ignorePatterns = []string{
		".git",
		".idea",
		".vscode",
		".cargo",
		"target",
		".forklift",
	}

	for _, pattern := range ignorePatterns {
		if pattern == entryName {
			return true
		}
	}

	return false
}
