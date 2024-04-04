package Compressors_test

import (
	"bytes"
	"forklift/CacheStorage/Compressors"
	"testing"
)

const data = `Lorem ipsum dolor sit amet, consectetur adipiscing elit, sed do eiusmod tempor incididunt
ut labore et dolore magna aliqua. Ut enim ad minim veniam, quis nostrud exercitation ullamco laboris
nisi ut aliquip ex ea commodo consequat. Duis aute irure dolor in reprehenderit in voluptate velit
esse cillum dolore eu fugiat nulla pariatur. Excepteur sint occaecat cupidatat non proident, sunt
in culpa qui officia deserunt mollit anim id est laborum.`

func TestZStdCompressor(t *testing.T) {
	var params = map[string]interface{}{}
	var compressor = Compressors.NewZStdCompressor(&params)

	var reader = bytes.NewReader([]byte(data))
	var compressed, _ = compressor.Compress(reader)

	var decompressed, _ = compressor.Decompress(compressed)
	var buf = bytes.Buffer{}
	buf.ReadFrom(decompressed)
	if string(buf.Bytes()) != data {
		t.Error("Data mismatch")
	}
}

func TestLzma2Compressor(t *testing.T) {
	var params = map[string]interface{}{}
	var compressor = Compressors.NewLzma2CCompressor(&params)

	var reader = bytes.NewReader([]byte(data))
	var compressed, _ = compressor.Compress(reader)

	var decompressed, _ = compressor.Decompress(compressed)
	var buf = bytes.Buffer{}
	buf.ReadFrom(decompressed)
	if string(buf.Bytes()) != data {
		t.Error("Data mismatch")
	}
}

func TestNoneCompressor(t *testing.T) {
	var compressor = &Compressors.NoneCompressor{}

	var reader = bytes.NewReader([]byte(data))
	var compressed, _ = compressor.Compress(reader)

	var decompressed, _ = compressor.Decompress(compressed)
	var buf = bytes.Buffer{}
	buf.ReadFrom(decompressed)
	if string(buf.Bytes()) != data {
		t.Error("Data mismatch")
	}
}
