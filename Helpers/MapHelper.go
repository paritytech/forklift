package Helpers

func MapGet[T string | int64 | bool](args *map[string]interface{}, key string, defaultValue T) T {
	var s, ok = (*args)[key]

	if ok {
		return s.(T)
	} else {
		return defaultValue
	}
}
