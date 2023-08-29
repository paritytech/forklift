package Storages

import (
	"bytes"
	"fmt"
	"forklift/CliTools"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"io"
	"log"
)

type S3Storage struct {
	session *session.Session
	bucket  string
}

func NewS3Storage(params *map[string]string) *S3Storage {
	s3s := S3Storage{
		bucket: CliTools.ExtractParam(params, "S3_BUCKET_NAME", "forklift", true),
	}

	var staticCreds = credentials.NewStaticCredentials(
		CliTools.ExtractParam(params, "S3_ACCESS_KEY_ID", "", true),
		CliTools.ExtractParam(params, "S3_SECRET_ACCESS_KEY", "", true),
		"",
	)

	s3s.session = session.Must(session.NewSession(&aws.Config{
		DisableSSL:       aws.Bool(CliTools.ExtractParam(params, "S3_USE_SSL", true, true)),
		Credentials:      staticCreds,
		Endpoint:         aws.String(CliTools.ExtractParam(params, "S3_ENDPOINT_URL", "", true)),
		Region:           aws.String("auto"),
		S3ForcePathStyle: aws.Bool(true),
	}))
	return &s3s
}

func (storage *S3Storage) Upload(key string, reader *io.Reader, metadata map[string]*string) {
	uploader := s3manager.NewUploader(storage.session)
	log.Println(storage.bucket)

	_, err := uploader.Upload(&s3manager.UploadInput{
		Bucket:   aws.String(storage.bucket),
		Key:      aws.String(key),
		Body:     *reader,
		Metadata: metadata,
	})
	if err != nil {
		log.Fatalf("failed to upload file %s\n%s", key, err)
	}
}

func (storage *S3Storage) Download(key string) io.Reader {
	downloader := s3manager.NewDownloader(storage.session)

	buf := aws.NewWriteAtBuffer([]byte{})

	n, err := downloader.Download(buf, &s3.GetObjectInput{
		Bucket: aws.String(storage.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		var _ = fmt.Errorf("failed to download file, %v", err)
	}
	if n == 0 {
		return nil
	}
	return bytes.NewBuffer(buf.Bytes())
}
