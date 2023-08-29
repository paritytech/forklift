package CliTools

import (
	"log"
	"os"
	"strconv"
)

func getByKey(args *map[string]string, key string, defaultFromEnvironment bool) (string, bool) {
	var s, ok = (*args)[key]

	if ok {
		return s, true
	}

	if defaultFromEnvironment {
		var fromEnv, ok = os.LookupEnv(key)
		if ok {
			return fromEnv, true
		}
	}

	return "", false
}

func ExtractParam[T string | int64 | bool](args *map[string]string, key string, defaultValue T, defaultFromEnvironment bool) T {
	var s, ok = getByKey(args, key, defaultFromEnvironment)

	if !ok {
		return defaultValue
	}

	var parsed any
	var err error

	switch any(s).(type) {
	case string:
		parsed = s
	case int64:
		var p, e = strconv.ParseInt(s, 10, 64)
		err = e
		parsed = p
	case bool:
		var p, e = strconv.ParseBool(s)
		err = e
		parsed = p
	}

	if err != nil {
		log.Printf("Failed to parse param `%s` with value `%s`\n", key, s)
		return defaultValue
	}

	return parsed.(T)
}
