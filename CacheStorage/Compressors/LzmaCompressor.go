package Compressors

import (
	"bytes"
	"github.com/ulikunitz/xz"
	"io"
)

type LzmaCompressor struct {
	ICompressor
}

func (compressor *LzmaCompressor) Compress(input io.Reader) (io.Reader, error) {
	var buf bytes.Buffer
	var writer, err = xz.NewWriter(&buf)
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

func (compressor *LzmaCompressor) Decompress(input io.Reader) (io.Reader, error) {
	var buf bytes.Buffer

	var reader, err = xz.NewReader(input)
	if err != nil {
		return nil, NewForkliftCompressorError("NewReader error", err)
	}

	_, err = io.Copy(&buf, reader)
	if err != nil {
		return nil, NewForkliftCompressorError("io.copy error", err)
	}

	return &buf, nil
}

func (compressor *LzmaCompressor) GetKey() string {
	return "xz"
}
