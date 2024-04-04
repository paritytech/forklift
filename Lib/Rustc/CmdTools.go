package Rustc

import (
	"path/filepath"
	"strings"
)

// GetExternDeps returns a list of external dependencies from argument list (--extern ...)
func GetExternDeps(args *[]string, basePathOnly bool) *[]string {
	var result []string

	for i := 0; i < len(*args); i++ {
		if (*args)[i] == "--extern" {

			var parts = strings.Split((*args)[i+1], "=")

			if len(parts) < 2 && basePathOnly {
				result = append(result, parts[0])
			} else if len(parts) < 2 && !basePathOnly {
			} else if basePathOnly {
				result = append(result, filepath.Base(parts[1]))
			} else {
				result = append(result, parts[1])
			}

			i++
		}
	}

	return &result
}
