package Compressors

import (
	"io"
)

type NoneCompressor struct {
}

func (n *NoneCompressor) Compress(input *io.Reader) io.Reader {
	return *input
}

func (n *NoneCompressor) Decompress(input *io.Reader) io.Reader {
	return *input
}
