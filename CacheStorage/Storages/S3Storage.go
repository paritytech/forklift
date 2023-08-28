package Storages

import (
	"bytes"
	"fmt"
	"io"
	"os"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
)

type S3Storage struct {
	session *session.Session
}

func NewS3Storage() *S3Storage {
	s3s := new(S3Storage)
	s3s.session = session.Must(session.NewSession(&aws.Config{
		DisableSSL:       aws.Bool(true),
		Credentials:      credentials.NewStaticCredentials(os.Getenv("S3_ACCESS_KEY_ID"), os.Getenv("S3_SECRET_ACCESS_KEY"), ""),
		Endpoint:         aws.String(os.Getenv("S3_ENDPOINT_URL")),
		Region:           aws.String("auto"),
		S3ForcePathStyle: aws.Bool(true),
	}))
	return s3s
}

func (storage S3Storage) Upload(key string, reader io.Reader) {
	uploader := s3manager.NewUploader(storage.session)

	var busketName = os.Getenv("S3_BUCKET_NAME")

	// Upload the file to S3.
	_, err := uploader.Upload(&s3manager.UploadInput{
		Bucket: aws.String(busketName),
		Key:    aws.String(key),
		Body:   reader,
	})
	if err != nil {
		var _ = fmt.Errorf("failed to upload file, %v", err)
	}
}

func (storage S3Storage) Download(key string) io.Reader {
	downloader := s3manager.NewDownloader(storage.session)

	var busketName = os.Getenv("S3_BUCKET_NAME")

	buf := aws.NewWriteAtBuffer([]byte{})

	downloader.Download(buf, &s3.GetObjectInput{
		Bucket: aws.String(busketName),
		Key:    aws.String(key),
	})

	return bytes.NewBuffer(buf.Bytes())
}
