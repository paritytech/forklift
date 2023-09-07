package Compressors

import (
	"bytes"
	log "github.com/sirupsen/logrus"
	"github.com/ulikunitz/xz"
	"io"
)

type LzmaCompressor struct {
	ICompressor
}

func (compressor *LzmaCompressor) Compress(input *io.Reader) io.Reader {
	var buf bytes.Buffer
	var writer, err = xz.NewWriter(&buf)
	if err != nil {
		log.Fatalf("NewWriter error %s\n", err)
	}

	var _, err2 = io.Copy(writer, *input)
	if err2 != nil {
		log.Fatalf("Copy error %s\n", err2)
	}

	if err := writer.Close(); err != nil {
		log.Fatalf("w.Close error %s\n", err)
	}

	return &buf
}

func (compressor *LzmaCompressor) Decompress(input *io.Reader) io.Reader {
	var buf bytes.Buffer

	var reader, err = xz.NewReader(*input)
	if err != nil {
		log.Fatalf("NewReader error %s\n", err)
	}

	var _, err2 = io.Copy(&buf, reader)
	if err2 != nil {
		log.Fatalf("Read error %s\n", err2)
	}

	return &buf
}

func (compressor *LzmaCompressor) GetKey() string {
	return "xz"
}
