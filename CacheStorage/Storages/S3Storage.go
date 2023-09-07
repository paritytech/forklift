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
	log "github.com/sirupsen/logrus"
	"io"
)

type S3Storage struct {
	session *session.Session
	bucket  string
	client  *s3.S3
}

func NewS3Storage(params *map[string]string) *S3Storage {
	s3s := S3Storage{}

	var bucketName = CliTools.ExtractParam(params, "BUCKET_NAME", "forklift", true)
	bucketName = CliTools.ExtractParam(params, "S3_BUCKET_NAME", bucketName, true)

	s3s.bucket = bucketName

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

func (storage *S3Storage) GetMetadata(key string) (map[string]*string, bool) {

	var head, err = storage.client.HeadObject(&s3.HeadObjectInput{
		Key:    &key,
		Bucket: &storage.bucket,
	})

	if err != nil {
		var awsErr awserr.Error
		if errors.As(err, &awsErr) {
			switch awsErr.Code() {
			case "NotFound":
				return nil, false
			case s3.ErrCodeNoSuchBucket:
				log.Printf("bucket %s does not exist\n", storage.bucket)
			case s3.ErrCodeNoSuchKey:
				log.Printf("object with key %s does not exist in bucket %s\n", key, storage.bucket)
			}
		} else {
			log.Fatalf("failed to get head for file %s\n%s", key, err)
		}
	}

	return head.Metadata, true
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
		log.Fatalf("failed to upload file %s\n%s\n", key, err)
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
		var _ = fmt.Errorf("failed to download file, %v\n", err)
	}
	if n == 0 {
		return nil
	}
	return bytes.NewBuffer(buf.Bytes())
}
