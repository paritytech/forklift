package Compressors

import "bytes"

type ICompressor interface {
	Compress(buffer bytes.Buffer) bytes.Buffer
	Decompress(buffer bytes.Buffer) bytes.Buffer
}
