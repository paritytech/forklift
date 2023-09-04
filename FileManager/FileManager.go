package FileManager

import (
	"archive/tar"
	"bytes"
	"crypto/sha1"
	"fmt"
	"hash"
	"io"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type TargetFsEntry struct {
	path     string
	basePath string
	info     fs.FileInfo
}

type CacheItem struct {
	Name    string
	Version string
	HashInt string
	Hash    string
}

func ParseCacheRequest() []CacheItem {

	var b, _ = os.ReadFile("./items.cache")
	str := string(b)
	var splitStrings = strings.Split(strings.ReplaceAll(str, "\r\n", "\n"), "\n")

	var result []CacheItem

	for i := range splitStrings {
		var itemParts = strings.Split(splitStrings[i], "|")
		if len(itemParts) < 4 {
			continue
		}
		result = append(result, CacheItem{
			Name:    strings.TrimSpace(itemParts[0]),
			Version: strings.TrimSpace(itemParts[1]),
			HashInt: strings.TrimSpace(itemParts[2]),
			Hash:    strings.TrimSpace(itemParts[3]),
		})
	}

	return result
}

func FindOpt(dir string, key string) []TargetFsEntry {
	var files []TargetFsEntry

	//dir = filepath.Clean(dir)

	var matches, _ = filepath.Glob(filepath.Join(dir, "*"+key+"*"))

	for _, match := range matches {
		log.Println(match)
		//if strings.Contains(filepath.Base(match), strings.ReplaceAll(key.Name, "-","_")+""+key.Hash) {
		var info, _ = os.Stat(match)
		var relPath, _ = filepath.Rel(dir, match)
		//var absPath, _ = filepath.Abs(path)
		targetFile := TargetFsEntry{
			path:     relPath,
			basePath: dir,
			info:     info,
		}
		files = append(files, targetFile)
		//}

		//return nil
	}

	return files
}

// Find ed
func Find(dir string, key string) []TargetFsEntry {
	var files []TargetFsEntry

	dir = filepath.Clean(dir)

	err := filepath.WalkDir(dir, func(path string, de fs.DirEntry, err error) error {

		if err != nil {
			fmt.Println(err)
			return nil
		}

		if strings.Contains(filepath.Base(path), key) {
			info, _ := de.Info()
			var relPath, _ = filepath.Rel(dir, path)
			targetFile := TargetFsEntry{
				path:     relPath,
				basePath: dir,
				info:     info,
			}
			files = append(files, targetFile)
		}

		if de.IsDir() && path != dir {
			return filepath.SkipDir
		}

		return nil
	})

	if err != nil {
		log.Fatal(err)
	}

	return files
}

// Tar Tarf dgsrfdg h
func Tar(fsEntries []TargetFsEntry) (io.Reader, hash.Hash) {

	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)

	var sha = sha1.New()

	for _, fsEntry := range fsEntries {

		if fsEntry.info.IsDir() {
			tarDirectory(tw, fsEntry, sha)
		} else {
			tarFile(tw, fsEntry, sha)
		}
	}

	if buf.Len() <= 0 {
		return nil, nil
	}

	return &buf, sha
}

func tarDirectory(tarWriter *tar.Writer, entryInfo TargetFsEntry, hash hash.Hash) {

	err := filepath.WalkDir(filepath.Join(entryInfo.basePath, entryInfo.path), func(path string, de fs.DirEntry, err error) error {

		if !de.IsDir() {
			var info, _ = de.Info()
			var relPath, _ = filepath.Rel(entryInfo.basePath, path)
			tarFile(
				tarWriter,
				TargetFsEntry{
					path:     relPath,
					basePath: entryInfo.basePath,
					info:     info,
				},
				hash,
			)
		}

		return nil
	})
	if err != nil {
		log.Fatal(err)
	}
}

func tarFile(tarWriter *tar.Writer, entryInfo TargetFsEntry, hash hash.Hash) {
	hdr := &tar.Header{
		Name:    entryInfo.path,
		Mode:    int64(entryInfo.info.Mode().Perm()),
		Size:    entryInfo.info.Size(),
		ModTime: entryInfo.info.ModTime(),
	}

	err := tarWriter.WriteHeader(hdr)
	if err != nil {
		log.Fatal(err)
	}

	f, err := os.Open(filepath.Join(entryInfo.basePath, entryInfo.path))
	if err != nil {
		log.Fatal(err)
	}

	var mw = io.MultiWriter(tarWriter, hash)
	_, err = io.Copy(mw, f)
	if err != nil {
		log.Fatal(err)
	}

	f.Close()
}

func UnTar(path string, reader io.Reader) {

	tr := tar.NewReader(reader)

	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break // End of archive
		}
		if err != nil {
			log.Fatal(err)
		}

		filePath := filepath.Join(path, hdr.Name)
		os.MkdirAll(filepath.Dir(filePath), 0777)

		f, err := os.Create(filePath)

		if err != nil {
			log.Fatal(err)
		}

		if _, err := io.Copy(f, tr); err != nil {
			log.Fatal(err)
		}

		f.Close()
		os.Chmod(filePath, os.FileMode(hdr.Mode))
		os.Chtimes(filePath, time.Now(), hdr.ModTime)
	}
}
