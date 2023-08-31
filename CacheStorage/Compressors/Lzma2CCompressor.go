package Compressors

import (
	"bytes"
	"forklift/CliTools"
	"github.com/jamespfennell/xz"
	"io"
	"log"
)

type Lzma2CCompressor struct {
	level int
}

func NewLzma2CCompressor(params *map[string]string) *Lzma2CCompressor {
	var compressionLevel = CliTools.ExtractParam[int64](params, "LZMA2_COMPRESSION_LEVEL", int64(6), true)
	return &Lzma2CCompressor{
		level: int(compressionLevel),
	}
}

func (compressor *Lzma2CCompressor) Compress(input *io.Reader) io.Reader {
	var buf = bytes.Buffer{}
	var writer = xz.NewWriterLevel(&buf, compressor.level)

	var _, err2 = io.Copy(writer, *input)
	if err2 != nil {
		log.Fatalf("Copy error %s", err2)
	}

	if err := writer.Close(); err != nil {
		log.Fatalf("w.Close error %s", err)
	}

	return &buf
}

func (compressor *Lzma2CCompressor) Decompress(input *io.Reader) io.Reader {
	var buf bytes.Buffer

	var reader = xz.NewReader(*input)

	var _, err2 = io.Copy(&buf, reader)
	if err2 != nil {
		log.Fatalf("Read error %s", err2)
	}

	return &buf
}
