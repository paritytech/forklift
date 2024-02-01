//go:build no_lzma2

package Compressors

import (
	"errors"
	"forklift/Lib/Config"
	log "github.com/sirupsen/logrus"
	"strings"
)

func GetCompressor(config Config.ForkliftConfig) (ICompressor, error) {

	var name = config.Compression.Type
	var params = &config.General.Params

	switch strings.ToLower(name) {
	case "none":
		var s = &NoneCompressor{}
		return s, nil
	case "xz":
		var s = &LzmaCompressor{}
		return s, nil
	case "zstd":
		var s = NewZStdCompressor(params)
		return s, nil
	default:
		log.Fatalf("unsupported compressor `%s`\n", name)
		return nil, errors.New("unsupported compressor")
	}
}
