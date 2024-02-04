package Helpers

import (
	"strings"
)

func MapGet[T string | int64 | bool](args *map[string]interface{}, key string, defaultValue T) T {
	var s, ok = (*args)[strings.ToLower(key)]

	if ok {
		return s.(T)
	} else {
		return defaultValue
	}
}
