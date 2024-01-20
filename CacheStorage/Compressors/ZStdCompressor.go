package Compressors

import (
	"bytes"
	"fmt"
	"forklift/CliTools"
	"github.com/klauspost/compress/zstd"
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

func (compressor *ZStdCompressor) Compress(input io.Reader) (io.Reader, error) {
	var buf bytes.Buffer
	var writer, err = zstd.NewWriter(&buf, zstd.WithEncoderLevel(zstd.EncoderLevelFromZstd(compressor.level)))
	if err != nil {
		return nil, NewForkliftCompressorError("NewWriter error", err)
	}

	_, err = io.Copy(writer, input)
	if err != nil {
		return nil, NewForkliftCompressorError("io.copy error", err)
	}

	if err = writer.Close(); err != nil {
		return nil, err
		//log.Fatalf("w.Close error %s\n", err)
	}

	return &buf, nil
}

func (compressor *ZStdCompressor) Decompress(input io.Reader) (io.Reader, error) {
	var buf bytes.Buffer

	var reader, err = zstd.NewReader(input)
	if err != nil {
		return nil, NewForkliftCompressorError("NewReader error", err)
	}

	_, err = io.Copy(&buf, reader)
	if err != nil {
		return nil, NewForkliftCompressorError("io.copy error", err)
	}

	return &buf, nil
}

func (compressor *ZStdCompressor) GetKey() string {
	return fmt.Sprintf("zstd-%d", compressor.level)
}
