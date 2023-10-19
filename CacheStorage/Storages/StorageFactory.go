package Storages

import (
	"fmt"
	"forklift/Lib"
	"strings"
)

func GetStorageDriver(config Lib.ForkliftConfig) (IStorage, error) {

	var name = config.Storage.Type
	var params = &config.General.Params

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
	case "null":
		var s = NewNullStorage()
		return s, nil
	default:
		return nil, fmt.Errorf("unsupported driver `%s`\n", name)
	}
}
