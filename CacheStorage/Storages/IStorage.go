package Storages

import (
	"io"
	"time"
)

type IStorage interface {
	Upload(key string, reader *io.Reader, metadata map[string]*string) (*UploadResult, error)
	Download(key string) (*DownloadResult, error)
	GetMetadata(key string) (map[string]*string, bool)
}

type DownloadResult struct {
	Data io.Reader
	StorageResult
}

type UploadResult struct {
	StorageResult
}

type StorageResult struct {
	BytesCount int64
	Duration   time.Duration
	SpeedBps   int64
}
