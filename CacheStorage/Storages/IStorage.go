package Storages

import (
	"io"
)

type IStorage interface {
	Upload(key string, reader *io.Reader, metadata map[string]*string) error
	Download(key string) (io.Reader, error)
	GetMetadata(key string) (map[string]*string, bool)
}
