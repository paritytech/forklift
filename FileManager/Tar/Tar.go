package Tar

import (
	"archive/tar"
	"bytes"
	"crypto/sha1"
	"forklift/FileManager/Models"
	log "github.com/sirupsen/logrus"
	"hash"
	"io"
	"io/fs"
	"os"
	"path/filepath"
)

// Pack -
func Pack(fsEntries []Models.TargetFsEntry) (io.Reader, hash.Hash) {

	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)

	var sha = sha1.New()

	for _, fsEntry := range fsEntries {

		if fsEntry.Info.IsDir() {
			PackDirectory(tw, fsEntry.Path, sha)
		} else {
			PackFile(tw, fsEntry, sha)
		}
	}

	if buf.Len() <= 0 {
		return nil, nil
	}

	return &buf, sha
}

func PackDirectory(tarWriter *tar.Writer, dirPath string, hash hash.Hash) {

	err := filepath.WalkDir(dirPath, func(path string, de fs.DirEntry, err error) error {
		var t, _ = os.Lstat(path)

		if t.Mode().IsRegular() {
			PackFile(
				tarWriter,
				Models.TargetFsEntry{
					Path: path,
					Info: t,
				},
				hash,
			)
		} else if t.Mode()&fs.ModeSymlink != 0 {
		}

		return nil
	})
	if err != nil {
		log.Fatalln(err)
	}
}

func PackFile(tarWriter *tar.Writer, entryInfo Models.TargetFsEntry, hash hash.Hash) {
	hdr := &tar.Header{
		Name:    entryInfo.Path,
		Mode:    int64(entryInfo.Info.Mode().Perm()),
		Size:    entryInfo.Info.Size(),
		ModTime: entryInfo.Info.ModTime(),
	}

	err := tarWriter.WriteHeader(hdr)
	if err != nil {
		log.Fatalln(err)
	}

	f, err := os.Open(entryInfo.Path)
	if err != nil {
		log.Fatalln(err)
	}

	var mw = io.MultiWriter(tarWriter, hash)
	_, err = io.Copy(mw, f)
	if err != nil {
		log.Fatalln(err)
	}

	err = f.Close()
	if err != nil {
		log.Fatalln(err)
	}
}

func UnPack(path string, reader io.Reader) {

	tr := tar.NewReader(reader)

	for {
		header, err := tr.Next()
		if err == io.EOF {
			break // End of archive
		}
		if err != nil {
			log.Fatalln(err)
		}

		filePath := filepath.Join(path, header.Name)
		os.MkdirAll(filepath.Dir(filePath), 0777)

		f, err := os.Create(filePath)

		if err != nil {
			log.Fatalln(err)
		}

		if _, err := io.Copy(f, tr); err != nil {
			log.Fatalln(err)
		}

		f.Close()
		os.Chmod(filePath, os.FileMode(header.Mode))
		os.Chtimes(filePath, header.ModTime, header.ModTime)
	}
}
