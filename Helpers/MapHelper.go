package Helpers

import (
	"reflect"
	"strconv"
)

func MapGet[T string | int64 | bool](args *map[string]interface{}, key string, defaultValue T) T {
	var s, ok = (*args)[key]

	if !ok {
		return defaultValue
	}

	if reflect.TypeOf(s) == reflect.TypeOf(defaultValue) {
		return s.(T)
	}

	var result any

	if reflect.TypeOf(s).Kind() == reflect.String {
		switch reflect.TypeOf(defaultValue).Kind() {
		case reflect.Int64:
			result, _ = strconv.ParseInt(s.(string), 10, 64)
		case reflect.Bool:
			result, _ = strconv.ParseBool(s.(string))
		default:
			result = s.(string)
		}

		return result.(T)
	}

	return defaultValue
}
