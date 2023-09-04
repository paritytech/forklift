package Storages

import (
	"io"
)

type IStorage interface {
	Upload(key string, reader *io.Reader, metadata map[string]*string)
	Download(key string) io.Reader
	GetMetadata(key string) (map[string]*string, bool)
}
