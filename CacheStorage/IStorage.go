package CacheStorage

import "bytes"

type IStorage interface {
	Upload(key string, buffer bytes.Buffer)
	Download(key string) []byte
}
