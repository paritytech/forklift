package Storages

import (
	"bytes"
	"forklift/CliTools"
	log "github.com/sirupsen/logrus"
	"io"
	"os"
	"path/filepath"
)

type FsStorage struct {
	dir string
}

func NewFsStorage(params *map[string]string) *FsStorage {
	fsStorage := FsStorage{
		dir: CliTools.ExtractParam(params, "FS_DIR", "", true),
	}
	return &fsStorage
}

func (storage *FsStorage) GetMetadata(key string) (map[string]*string, bool) {
	var _, err = os.Stat(filepath.Join(storage.dir, key))
	return nil, err == nil
}

func (storage *FsStorage) Upload(key string, reader *io.Reader, _ map[string]*string) error {

	var file, err = os.Create(filepath.Join(storage.dir, key))
	if err != nil {
		log.Fatalln("Unable to create file", err)
	}
	defer file.Close()

	_, err2 := io.Copy(file, *reader)
	if err2 != nil {
		return err
	}

	return nil
}

func (storage *FsStorage) Download(key string) (io.Reader, error) {

	var path = filepath.Join(storage.dir, key)
	var _, errStat = os.Stat(path)
	if errStat != nil {
		return nil, nil
	}

	var file, err = os.Open(path)
	if err != nil {
		log.Errorf("Unable to open file", err)
		return nil, err
	}
	defer file.Close()

	var buf bytes.Buffer

	_, err2 := io.Copy(&buf, file)
	if err2 != nil {
		log.Errorf("Unable to read file", err)
		return nil, err2
	}

	return &buf, nil
}
