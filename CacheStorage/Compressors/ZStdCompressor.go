package Compressors

import (
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
	var pr, pw = io.Pipe()
	var writer, err = zstd.NewWriter(pw, zstd.WithEncoderLevel(zstd.EncoderLevelFromZstd(compressor.level)))
	if err != nil {
		return nil, NewForkliftCompressorError("NewWriter error", err)
	}

	// Start a goroutine to compress data
	go func() {
		defer writer.Close()
		defer pw.Close()

		_, err := io.Copy(writer, input)
		if err != nil {
			pw.CloseWithError(NewForkliftCompressorError("io.copy error", err))
			return
		}

		// Ensure writer is properly closed
		if err = writer.Close(); err != nil {
			pw.CloseWithError(err)
		}
	}()

	return pr, nil
}

func (compressor *ZStdCompressor) Decompress(input io.Reader) (io.Reader, error) {
	var pr, pw = io.Pipe()
	var reader, err = zstd.NewReader(input)
	if err != nil {
		return nil, NewForkliftCompressorError("NewReader error", err)
	}

	go func() {
		defer reader.Close()
		defer pw.Close()

		_, err := io.Copy(pw, reader)
		if err != nil {
			pw.CloseWithError(NewForkliftCompressorError("io.Copy error", err))
			return
		}
	}()

	return pr, nil
}

func (compressor *ZStdCompressor) GetKey() string {
	return fmt.Sprintf("zstd-%d", compressor.level)
}
