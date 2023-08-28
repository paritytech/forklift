package Compressors

import (
	"bytes"
	"io"
	"log"

	"github.com/ulikunitz/xz"
)

type LzmaCompressor struct {
}

func (l LzmaCompressor) Compress(input bytes.Buffer) *bytes.Buffer {
	var buf = bytes.Buffer{}
	var writer, err = xz.NewWriter(&buf)
	if err != nil {
		log.Fatalf("NewWriter error %s", err)
	}

	var _, err2 = writer.Write(input.Bytes())
	if err2 != nil {
		log.Fatalf("Write error %s", err)
	}

	return &buf
}

func (l LzmaCompressor) Decompress(input bytes.Buffer) *bytes.Buffer {
	var buf = bytes.Buffer{}

	var reader, err = xz.NewReader(&input)
	if err != nil {
		log.Fatalf("NewReader error %s", err)
	}

	var _, err2 = io.Copy(&buf, reader)
	if err2 != nil {
		log.Fatalf("Read error %s", err)
	}

	return &buf
}
