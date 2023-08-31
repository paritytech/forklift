package Compressors

import (
	"fmt"
	"strings"
)

func GetCompressor(name string, params *map[string]string) (ICompressor, error) {

	switch strings.ToLower(name) {
	case "none":
		var s = &NoneCompressor{}
		return s, nil
	case "xz":
		var s = &LzmaCompressor{}
		return s, nil
	case "lzma2":
		var s = NewLzma2CCompressor(params)
		return s, nil
	default:
		return nil, fmt.Errorf("unsupported compessor `%s`", name)
	}
}
