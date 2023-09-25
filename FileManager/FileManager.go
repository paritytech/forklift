package FileManager

import (
	"forklift/FileManager/Models"
	"forklift/Lib"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
)

func ParseCacheRequest() []Models.CacheItem {

	var b, _ = os.ReadFile("./items.cache")
	str := string(b)
	var splitStrings = strings.Split(strings.ReplaceAll(str, "\r\n", "\n"), "\n")

	var result []Models.CacheItem

	for i := range splitStrings {
		var itemParts = strings.Split(splitStrings[i], "|")
		if len(itemParts) < 4 {
			continue
		}
		result = append(result, Models.CacheItem{
			Name:    strings.TrimSpace(itemParts[0]),
			Version: strings.TrimSpace(itemParts[1]),
			HashInt: strings.TrimSpace(itemParts[2]),
			Hash:    strings.TrimSpace(itemParts[3]),
		})
	}

	return result
}

func FindAll(cacheItems []Models.CacheItem, dir string) map[string]*[]Models.TargetFsEntry {
	var dict map[string]*[]Models.TargetFsEntry = make(map[string]*[]Models.TargetFsEntry)

	for _, item := range cacheItems {
		dict[item.Hash] = &[]Models.TargetFsEntry{}
	}

	findAllRecursive(&dict, dir)
	return dict
}

var hashRegex = regexp.MustCompile(`[a-zA-Z0-9_]*-([a-fA-F0-9]{16})`)

func findAllRecursive(cacheItemsMap *map[string]*[]Models.TargetFsEntry, dir string) {
	var files, _ = os.ReadDir(dir)

	for _, file := range files {
		var filePath = filepath.Join(dir, file.Name())

		var matches = hashRegex.FindAllStringSubmatch(file.Name(), -1)
		var ok = false
		var result *[]Models.TargetFsEntry

		if len(matches) > 0 {
			var hash = matches[0][1]
			result, ok = (*cacheItemsMap)[hash]
		}

		if ok {
			if file.Type().IsRegular() {
				var info, _ = file.Info()
				targetFile := Models.TargetFsEntry{
					Path: filePath,
					Info: info,
				}
				*result = append(*result, targetFile)
			} else if file.Type().IsDir() {
				filepath.WalkDir(filePath, func(path string, d fs.DirEntry, err error) error {

					var info, _ = d.Info()
					if info.Mode().IsRegular() {
						targetFile := Models.TargetFsEntry{
							Path: path,
							Info: info,
						}
						*result = append(*result, targetFile)
					}
					return nil
				})
			}
		} else if file.IsDir() {
			findAllRecursive(cacheItemsMap, filePath)
		}
	}

}

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

func UploadDir(dir string) {

	var queue = make(chan struct {
		info fs.FileInfo
		path string
		i    int
	}, 20)

	var cpuCount = runtime.NumCPU()

	go func() {
		var dirEntries, _ = os.ReadDir(dir)

		for i, file := range dirEntries {
			var path = filepath.Join(dir, file.Name())
			var fileInfo, _ = file.Info()

			queue <- struct {
				info fs.FileInfo
				path string
				i    int
			}{info: fileInfo, path: path, i: i}
		}
		close(queue)
	}()

	Lib.Parallel(
		queue,
		cpuCount,
		func(obj struct {
			info fs.FileInfo
			path string
			i    int
		}) {

		})
}
