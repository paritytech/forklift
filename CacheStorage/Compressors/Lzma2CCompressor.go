//go:build !no_lzma2

package Compressors

import (
	"bytes"
	"fmt"
	"forklift/Helpers"
	"github.com/jamespfennell/xz"
	"io"
)

type Lzma2CCompressor struct {
	ICompressor
	level int
}

func NewLzma2CCompressor(params *map[string]interface{}) *Lzma2CCompressor {
	var compressionLevel = Helpers.MapGet[int64](params, "compressionLevel", int64(6))
	return &Lzma2CCompressor{
		level: int(compressionLevel),
	}
}

func (compressor *Lzma2CCompressor) Compress(input io.Reader) (io.Reader, error) {
	var buf = bytes.Buffer{}
	var writer = xz.NewWriterLevel(&buf, compressor.level)

	var _, err = io.Copy(writer, input)
	if err != nil {
		return nil, NewForkliftCompressorError("io.copy error", err)
	}

	if err = writer.Close(); err != nil {
		return nil, NewForkliftCompressorError("writer.Close error", err)
	}

	return &buf, nil
}

func (compressor *Lzma2CCompressor) Decompress(input io.Reader) (io.Reader, error) {
	var buf bytes.Buffer

	var reader = xz.NewReader(input)

	var _, err = io.Copy(&buf, reader)
	if err != nil {
		return nil, NewForkliftCompressorError("io.copy error", err)
	}

	return &buf, nil
}

func (compressor *Lzma2CCompressor) GetKey() string {
	return fmt.Sprintf("lzma2-%d", compressor.level)
}
