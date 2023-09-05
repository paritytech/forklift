//go:build no_lzma2

package Compressors

import (
	"errors"
	log "github.com/sirupsen/logrus"
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
	default:
		log.Fatalf("unsupported compressor `%s`\n", name)
		return nil, errors.New("unsupported compressor")
	}
}
