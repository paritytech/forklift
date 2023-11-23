package FileManager

import (
	"crypto/sha1"
	"fmt"
	"forklift/FileManager/Models"
	"forklift/Lib"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
)

func MergeCacheRequest() {

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
			if len(itemParts) < 7 {
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
			})
		}
	}

	return result
}

func GetCheckSum(files []string, baseDir string) string {

	var sha = sha1.New()

	for _, file := range files {
		var path string
		if baseDir != "" {
			path = filepath.Join(baseDir, file)
		} else {
			path = file
		}

		var reader, _ = os.Open(path)

		io.Copy(sha, reader)

		reader.Close()
	}

	return fmt.Sprintf("%x", sha.Sum(nil))
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
