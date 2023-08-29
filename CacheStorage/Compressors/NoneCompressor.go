package Compressors

import (
	"bytes"
	"io"
)

type NoneCompressor struct {
}

func (n *NoneCompressor) Compress(input *io.Reader) io.Reader {
	var buf bytes.Buffer
	io.Copy(&buf, *input)
	return &buf
}

func (n *NoneCompressor) Decompress(input *io.Reader) io.Reader {
	var buf bytes.Buffer
	io.Copy(&buf, *input)
	return &buf
}
