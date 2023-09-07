package Storages

import (
	"fmt"
	"strings"
)

func GetStorageDriver(name string, params *map[string]string) (IStorage, error) {
	switch strings.ToLower(name) {
	case "s3":
		var s = NewS3Storage(params)
		return s, nil
	case "gcs":
		var s = NewGcsStorage(params)
		return s, nil
	case "fs":
		var s = NewFsStorage(params)
		return s, nil
	default:
		return nil, fmt.Errorf("unsupported driver `%s`\n", name)
	}
}
