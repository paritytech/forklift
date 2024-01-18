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

func (storage *NullStorage) Upload(key string, reader *io.Reader, _ map[string]*string) error {
	return nil
}

func (storage *NullStorage) Download(key string) (io.Reader, error) {
	return nil, nil
}
