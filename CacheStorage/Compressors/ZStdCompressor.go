package Compressors

import (
	"bytes"
	"fmt"
	"forklift/Helpers"
	"github.com/klauspost/compress/zstd"
	"io"
)

type ZStdCompressor struct {
	ICompressor
	level int
}

func NewZStdCompressor(params *map[string]interface{}) *ZStdCompressor {
	var compressionLevel = Helpers.MapGet[int64](params, "compressionLevel", int64(3))
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

	_, err = writer.ReadFrom(input)
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

	_, err = reader.WriteTo(&buf)
	if err != nil {
		return nil, NewForkliftCompressorError("reader.WriteTo error", err)
	}

	return &buf, nil
}

func (compressor *ZStdCompressor) GetKey() string {
	return fmt.Sprintf("zstd-%d", compressor.level)
}
