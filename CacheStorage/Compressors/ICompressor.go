package Compressors

import (
	"io"
)

type ICompressor interface {
	Compress(input *io.Reader) io.Reader
	Decompress(input *io.Reader) io.Reader
	GetKey() string
}
