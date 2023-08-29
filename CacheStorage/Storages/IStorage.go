package Storages

import (
	"io"
)

type IStorage interface {
	Upload(key string, reader *io.Reader)
	Download(key string) io.Reader
}
