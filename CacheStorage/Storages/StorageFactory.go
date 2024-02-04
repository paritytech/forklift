package Storages

import (
	"fmt"
	"forklift/Lib/Config"
	"strings"
)

func GetStorageDriver(config Config.ForkliftConfig) (IStorage, error) {

	var name = config.Storage.Type

	switch strings.ToLower(name) {
	case "s3":
		var s = NewS3Storage(config.GetMap("storage.s3"))
		return s, nil
	case "gcs":
		var s = NewGcsStorage(config.GetMap("storage.gcs"))
		return s, nil
	case "fs":
		var s = NewFsStorage(config.GetMap("storage.fs"))
		return s, nil
	case "null":
		var s = NewNullStorage()
		return s, nil
	default:
		return nil, fmt.Errorf("unsupported driver `%s`\n", name)
	}
}
