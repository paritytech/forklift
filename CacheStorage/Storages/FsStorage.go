package Storages

import (
	"bytes"
	"encoding/json"
	"forklift/Helpers"
	"forklift/Lib/Diagnostic/Time"
	log "forklift/Lib/Logging/ConsoleLogger"
	"io"
	"os"
	"path/filepath"
)

type FsStorage struct {
	dir string
}

func NewFsStorage(params *map[string]interface{}) *FsStorage {
	fsStorage := FsStorage{
		dir: Helpers.MapGet(params, "directory", ""),
	}
	return &fsStorage
}

func (storage *FsStorage) GetMetadata(key string) (map[string]*string, bool) {
	var _, err = os.Stat(filepath.Join(storage.dir, key))
	if err != nil {
		return nil, false
	}

	metaPath := filepath.Join(storage.dir, key+".meta")
	metaData, err := os.ReadFile(metaPath)
	if err != nil {
		// File exists but no metadata sidecar
		return nil, true
	}

	var metadata map[string]*string
	if err := json.Unmarshal(metaData, &metadata); err != nil {
		return nil, true
	}

	return metadata, true
}

func (storage *FsStorage) Upload(key string, reader io.Reader, metadata map[string]*string) (*UploadResult, error) {

	// Write metadata sidecar file if metadata is provided
	if len(metadata) > 0 {
		metaData, err := json.Marshal(metadata)
		if err == nil {
			os.WriteFile(filepath.Join(storage.dir, key+".meta"), metaData, 0644)
		}
	}


	var file, err = os.Create(filepath.Join(storage.dir, key))
	if err != nil {
		log.Fatalf("Unable to create file, %s", err)
		return nil, err
	}
	defer file.Close()

	var timer = Time.NewForkliftTimer()

	timer.Start("write")
	n, err2 := io.Copy(file, reader)
	if err2 != nil {
		return nil, err
	}
	var duration = timer.Stop("write")

	return &UploadResult{
		StorageResult{
			BytesCount: n,
			Duration:   duration,
			SpeedBps:   int64(float64(n) / duration.Seconds()),
		},
	}, nil
}

func (storage *FsStorage) Download(key string) (*DownloadResult, error) {

	var timer = Time.NewForkliftTimer()

	var path = filepath.Join(storage.dir, key)
	var fileInfo, errStat = os.Stat(path)
	if errStat != nil {
		return nil, nil
	}

	var file, err = os.Open(path)
	if err != nil {
		log.Errorf("Unable to open file: %s", err)
		return nil, err
	}
	defer file.Close()

	var buf = bytes.NewBuffer(make([]byte, 0, fileInfo.Size()))

	timer.Start("read")
	bytesWritten, err2 := io.Copy(buf, file)
	var duration = timer.Stop("read")

	if err2 != nil {
		log.Errorf("Unable to read file: %s", err)
		return nil, err2
	}

	var result = DownloadResult{
		Data: buf,
	}

	result.BytesCount = bytesWritten
	result.Duration = duration
	result.SpeedBps = int64(float64(bytesWritten) / duration.Seconds())

	return &result, nil
}
