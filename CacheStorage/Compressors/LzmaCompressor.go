package Compressors

import (
	"bytes"
	"io"
	"log"

	"github.com/ulikunitz/xz"
)

type LzmaCompressor struct {
}

func (l *LzmaCompressor) Compress(input *io.Reader) io.Reader {
	var buf bytes.Buffer
	var writer, err = xz.NewWriter(&buf)
	if err != nil {
		log.Fatalf("NewWriter error %s", err)
	}

	var _, err2 = io.Copy(writer, *input)
	if err2 != nil {
		log.Fatalf("Copy error %s", err2)
	}

	if err := writer.Close(); err != nil {
		log.Fatalf("w.Close error %s", err)
	}

	return &buf
}

func (l *LzmaCompressor) Decompress(input *io.Reader) io.Reader {
	var buf bytes.Buffer

	var reader, err = xz.NewReader(*input)
	if err != nil {
		log.Fatalf("NewReader error %s", err)
	}

	var _, err2 = io.Copy(&buf, reader)
	if err2 != nil {
		log.Fatalf("Read error %s", err2)
	}

	return &buf
}

func (l *LzmaCompressor) GetKey() string {
	return "xz"
}
