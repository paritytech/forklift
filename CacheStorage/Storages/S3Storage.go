package Storages

import (
	"bytes"
	"errors"
	"fmt"
	"forklift/CliTools"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"io"
	"log"
	"os"
)

type S3Storage struct {
	session *session.Session
	bucket  string
	client  *s3.S3
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

	s3s.client = s3.New(s3s.session, &aws.Config{})

	return &s3s
}

func (storage *S3Storage) GetMetadata(key string) map[string]*string {

	var head, err = storage.client.HeadObject(&s3.HeadObjectInput{
		Key:    &key,
		Bucket: &storage.bucket,
	})

	if err != nil {
		var aerr awserr.Error
		if errors.As(err, &aerr) {
			switch aerr.Code() {
			case "NotFound":
				log.Println("NotFound", key)
				return nil
			case s3.ErrCodeNoSuchBucket:
				log.Println("bucket %s does not exist", os.Args[1])
			case s3.ErrCodeNoSuchKey:
				log.Println("object with key %s does not exist in bucket %s", os.Args[2], os.Args[1])
			}
		} else {
			log.Fatalf("failed to get head for file %s\n%s", key, err)
		}
	}

	return head.Metadata
}

func (storage *S3Storage) Upload(key string, reader *io.Reader, metadata map[string]*string) {
	uploader := s3manager.NewUploader(storage.session)

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
