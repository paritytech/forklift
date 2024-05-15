package Storages

import (
	"bytes"
	"io"
	"math/rand"
	"strings"
	"testing"
	"time"
)

const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

var seededRand *rand.Rand = rand.New(rand.NewSource(time.Now().UnixNano()))

func randomString(length int) string {
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[seededRand.Intn(len(charset))]
	}
	return string(b)
}

func createStorage() *S3Storage {
	return NewS3Storage(&map[string]interface{}{
		"bucketName":  "forklift",
		"useSsl":      false,
		"endpointUrl": "http://192.168.10.12:9000",
		"concurrency": int64(1),
	})
}

func TestS3Storage(t *testing.T) {

	var storage = createStorage()

	var key = randomString(10)

	t.Run("Upload", func(t *testing.T) {
		var buffer = bytes.NewBuffer([]byte("hello, upload"))
		_, err := storage.Upload("key", buffer, nil)
		if err != nil {
			t.Error("Upload failed", err)
		}
	})

	t.Run("Download", func(t *testing.T) {
		result, err := storage.Download("key")
		if err != nil {
			t.Error("Download failed", err)
		}

		if result == nil {
			t.Error("Download failed, key does not exist")
		}

		var builder = strings.Builder{}
		io.Copy(&builder, result.Data)

		if builder.String() != "hello, upload" {
			t.Fatalf("Download failed, content is not equal")
		}
	})

	t.Run("Not_found", func(t *testing.T) {
		result, err := storage.Download(key + "123")
		if err != nil {
			t.Fatalf("Not_found failed with error: %s", err)
		}

		if !(err == nil && result == nil) {
			t.Fatalf("Not_found failed, %s, %v", err, result)
		}
	})
}
