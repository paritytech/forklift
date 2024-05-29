package Storages

import (
	"io"
)

// NullStorage Do nothing, for test purposes
type NullStorage struct {
	IStorage
}

func NewNullStorage() *NullStorage {
	fsStorage := NullStorage{}
	return &fsStorage
}

func (storage *NullStorage) GetMetadata(_ string) (map[string]*string, bool) {
	return nil, false
}

func (storage *NullStorage) Upload(_ string, _ io.Reader, _ map[string]*string) (*UploadResult, error) {
	return &UploadResult{
		StorageResult{
			BytesCount: 0,
			Duration:   0,
			SpeedBps:   0,
		},
	}, nil
}

func (storage *NullStorage) Download(string) (*DownloadResult, error) {
	return nil, nil
}
