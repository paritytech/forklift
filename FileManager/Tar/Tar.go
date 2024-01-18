package Tar

import (
	"archive/tar"
	"bytes"
	"crypto/sha1"
	log "github.com/sirupsen/logrus"
	"hash"
	"io"
	"io/fs"
	"os"
	"path/filepath"
)

// Pack - Pack forklift tar archive
func Pack(fsEntries []string) (io.Reader, hash.Hash) {

	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)

	var sha = sha1.New()

	for _, fsEntry := range fsEntries {

		log.Tracef("packing %s", fsEntry)
		var info, e = os.Stat(fsEntry)
		if e != nil {
			log.Error(e)
		}
		if info.IsDir() {
			PackDirectory(tw, fsEntry, sha)
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
				path,
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

func PackFile(tarWriter *tar.Writer, path string, hash hash.Hash) {

	var info, e = os.Stat(path)

	if e != nil {
		log.Errorf("file stat error")
	}

	hdr := &tar.Header{
		Name:    path,
		Mode:    int64(info.Mode().Perm()),
		Size:    info.Size(),
		ModTime: info.ModTime(),
	}

	err := tarWriter.WriteHeader(hdr)
	if err != nil {
		log.Fatalln(err)
	}

	f, err := os.Open(path)
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

// UnPack -	Unpack forklift tar archive
func UnPack(path string, reader io.Reader) error {

	tr := tar.NewReader(reader)

	for {
		header, err := tr.Next()
		if err == io.EOF {
			break // End of archive
		}
		if err != nil {
			log.Errorf("failed to read tar header: %s", err)
			return err
		}

		filePath := filepath.Join(path, header.Name)

		err = os.MkdirAll(filepath.Dir(filePath), 0777)
		if err != nil {
			log.Errorf("UnPack MkdirAll %s", err)
			return err
		}

		f, err := os.Create(filePath)

		if err != nil {
			log.Errorf("UnPack os.Create error: %s", err)
			return err
		}

		if w, err := io.Copy(f, tr); err != nil {
			log.Errorf("UnPack io.Copy error: %s", err)
			return err
		} else {
			log.Tracef("Unpacked %s written: %d", filePath, w)
		}

		f.Close()
		os.Chmod(filePath, 0777)
	}

	return nil
}
