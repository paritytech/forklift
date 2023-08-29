package FileManager

import (
	"archive/tar"
	"bytes"
	"fmt"
	"io"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strings"
)

type TargetFsEntry struct {
	path string
	//absPath  string
	basePath string
	info     fs.FileInfo
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
			//var absPath, _ = filepath.Abs(path)
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
func Tar(fsEntries []TargetFsEntry) io.Reader {

	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)

	for _, fsEntry := range fsEntries {

		if fsEntry.info.IsDir() {
			tarDirectory(tw, fsEntry)
		} else {
			tarFile(tw, fsEntry)
		}
	}

	return &buf
}

func tarDirectory(tarWriter *tar.Writer, entryInfo TargetFsEntry) {

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
			)
		}

		return nil
	})
	if err != nil {
		log.Fatal(err)
	}
}

func tarFile(tarWriter *tar.Writer, entryInfo TargetFsEntry) {
	hdr := &tar.Header{
		Name: entryInfo.path,
		Mode: 0777,
		Size: entryInfo.info.Size(),
	}

	err := tarWriter.WriteHeader(hdr)
	if err != nil {
		log.Fatal(err)
	}

	f, err := os.Open(filepath.Join(entryInfo.basePath, entryInfo.path))
	if err != nil {
		log.Fatal(err)
	}
	body := make([]byte, entryInfo.info.Size())
	f.Read(body)

	_, err = tarWriter.Write(body)
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
	}
}
