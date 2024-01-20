package Compressors

import (
	"fmt"
	"io"
)

type ICompressor interface {
	Compress(input io.Reader) (io.Reader, error)
	Decompress(input io.Reader) (io.Reader, error)
	GetKey() string
}

type ForkliftCompressorError struct {
	error
	Message string
	Err     error
}

func NewForkliftCompressorError(message string, err error) *ForkliftCompressorError {
	return &ForkliftCompressorError{
		Message: message,
		Err:     err,
	}
}

func (fce *ForkliftCompressorError) String() string {
	return fmt.Sprintf("Compressor error: %s, inner error: %v", fce.Message, fce.Err)
}
