package FileManager

import (
	"forklift/FileManager/Models"
	"log"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
)

func GetTrueRelFilePath(workDir string, path string) string {
	if workDir == "" {
		workDir, _ = os.Getwd()
	}

	var absPath, err = filepath.Abs(path)
	if err != nil {
		log.Fatalln(err)
	}

	absPath, err = filepath.EvalSymlinks(absPath)
	if err != nil {
		log.Fatalln(err)
	}

	relPath, err := filepath.Rel(workDir, absPath)
	if err != nil {
		log.Fatalln(err)
	}

	return relPath
}

func ParseCacheRequest() []Models.CacheItem {

	var files, _ = filepath.Glob(path.Join(".forklift", "items-cache", "item-*"))

	var result []Models.CacheItem

	for _, file := range files {

		var b, _ = os.ReadFile(file)

		str := string(b)
		var splitStrings = strings.Split(strings.ReplaceAll(str, "\r\n", "\n"), "\n")

		for i := range splitStrings {
			var itemParts = strings.Split(splitStrings[i], "|")
			if len(itemParts) < 8 {
				continue
			}
			result = append(result, Models.CacheItem{
				Name:                strings.TrimSpace(itemParts[0]),
				Version:             strings.TrimSpace(itemParts[1]),
				HashInt:             strings.TrimSpace(itemParts[2]),
				Hash:                strings.TrimSpace(itemParts[3]),
				CachePackageName:    strings.TrimSpace(itemParts[4]),
				OutDir:              strings.TrimSpace(itemParts[5]),
				CrateSourceChecksum: strings.TrimSpace(itemParts[6]),
				RustCArgsHash:       strings.TrimSpace(itemParts[7]),
				//CrateExternDepsChecksum:   strings.TrimSpace(itemParts[8]),
			})
		}
	}

	return result
}

var hashRegex = regexp.MustCompile(`[a-zA-Z0-9_]*-([a-fA-F0-9]{16})`)

func Find(dir string, key string, recursive bool) []Models.TargetFsEntry {
	var result []Models.TargetFsEntry

	if recursive {
		result = findRecursive(dir, key, false)
	} else {
		var files, _ = os.ReadDir(dir)
		for _, file := range files {
			var filePath = filepath.Join(dir, file.Name())

			if strings.Contains(file.Name(), key) {
				if file.Type().IsRegular() {
					var info, _ = file.Info()
					targetFile := Models.TargetFsEntry{
						Path: filePath,
						Info: info,
					}
					result = append(result, targetFile)
				}
			}
		}
	}

	return result
}

func findRecursive(dir string, key string, all bool) []Models.TargetFsEntry {
	var files, _ = os.ReadDir(dir)

	var result []Models.TargetFsEntry

	for _, file := range files {
		var filePath = filepath.Join(dir, file.Name())

		if strings.Contains(file.Name(), key) || all {
			if file.Type().IsRegular() {
				var info, _ = file.Info()
				targetFile := Models.TargetFsEntry{
					Path: filePath,
					Info: info,
				}
				result = append(result, targetFile)
			} else if file.Type().IsDir() {
				result = append(result, findRecursive(filePath, key, true)...)
			}

		} else if file.IsDir() {
			result = append(result, findRecursive(filePath, key, false)...)
		}
	}

	return result
}
