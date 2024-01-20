package Compressors

import (
	"io"
)

type NoneCompressor struct {
	ICompressor
}

func (compressor *NoneCompressor) Compress(input io.Reader) (io.Reader, error) {
	return input, nil
}

func (compressor *NoneCompressor) Decompress(input io.Reader) (io.Reader, error) {
	return input, nil
}

func (compressor *NoneCompressor) GetKey() string {
	return "none"
}
