package Compressors

import (
	"bytes"
	"fmt"
	"forklift/CliTools"
	"github.com/klauspost/compress/zstd"
	log "github.com/sirupsen/logrus"
	"io"
)

type ZStdCompressor struct {
	ICompressor
	level int
}

func NewZStdCompressor(params *map[string]string) *ZStdCompressor {
	var compressionLevel = CliTools.ExtractParam[int64](params, "COMPRESSION_LEVEL", int64(3), true)
	compressionLevel = CliTools.ExtractParam[int64](params, "ZSTD_COMPRESSION_LEVEL", compressionLevel, true)
	return &ZStdCompressor{
		level: int(compressionLevel),
	}
}

func (compressor *ZStdCompressor) Compress(input *io.Reader) io.Reader {
	var buf bytes.Buffer
	var writer, err = zstd.NewWriter(&buf, zstd.WithEncoderLevel(zstd.EncoderLevelFromZstd(compressor.level)))
	if err != nil {
		log.Fatalf("NewWriter error %s\n", err)
	}

	var _, err2 = io.Copy(writer, *input)
	if err2 != nil {
		log.Fatalf("Copy error %s\n", err2)
	}

	if err := writer.Close(); err != nil {
		log.Fatalf("w.Close error %s\n", err)
	}

	return &buf
}

func (compressor *ZStdCompressor) Decompress(input *io.Reader) io.Reader {
	var buf bytes.Buffer

	var reader, err = zstd.NewReader(*input)
	if err != nil {
		log.Fatalf("NewReader error %s\n", err)
	}

	var _, err2 = io.Copy(&buf, reader)
	if err2 != nil {
		log.Fatalf("Read error %s\n", err2)
	}

	return &buf
}

func (compressor *ZStdCompressor) GetKey() string {
	return fmt.Sprintf("zstd-%d", compressor.level)
}
