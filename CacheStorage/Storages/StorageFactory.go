package Storages

import (
	"fmt"
	"strings"
)

func GetStorageDriver(name string) (IStorage, error) {
	switch strings.ToLower(name) {
	case "s3":
		var s = NewS3Storage()
		return s, nil
	default:
		return nil, fmt.Errorf("unsupported driver `%s`", name)
	}
}
