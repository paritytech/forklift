package Storages

import (
	"bytes"
	"errors"
	"forklift/Helpers"
	"forklift/Lib/Diagnostic/Time"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	log "github.com/sirupsen/logrus"
	"io"
	"strings"
)

type S3Storage struct {
	session     *session.Session
	bucket      string
	client      *s3.S3
	concurrency int
}

func NewS3Storage(params *map[string]interface{}) *S3Storage {
	s3s := S3Storage{}

	s3s.concurrency = int(Helpers.MapGet[int64](params, "concurrency", 1))

	var bucketName = Helpers.MapGet(params, "bucketName", "forklift")
	s3s.bucket = bucketName

	var staticCredentials = credentials.NewStaticCredentials(
		Helpers.MapGet(params, "accessKeyId", ""),
		Helpers.MapGet(params, "secretAccessKey", ""),
		"",
	)

	s3s.session = session.Must(session.NewSession(&aws.Config{
		DisableSSL:       aws.Bool(Helpers.MapGet(params, "useSsl", true)),
		Credentials:      staticCredentials,
		Endpoint:         aws.String(Helpers.MapGet(params, "endpointUrl", "")),
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
				log.Tracef("bucket %s does not exist", storage.bucket)
			case s3.ErrCodeNoSuchKey:
				log.Tracef("object with key %s does not exist in bucket %s", key, storage.bucket)
			}
		} else {
			log.Fatalf("failed to get head for file %s, %s", key, err)
		}
	}

	var metadata = make(map[string]*string, len(head.Metadata))

	for key, value := range head.Metadata {
		metadata[strings.ToLower(key)] = value
	}

	return metadata, true
}

func (storage *S3Storage) Upload(key string, reader *io.Reader, metadata map[string]*string) (*UploadResult, error) {
	uploader := s3manager.NewUploader(storage.session)

	var buf = bytes.Buffer{}
	var n, _ = io.Copy(&buf, *reader)

	var normalizedMetadata = make(map[string]*string, len(metadata))
	for key, value := range metadata {
		normalizedMetadata[strings.ToLower(key)] = value
	}

	var timer = Time.NewForkliftTimer()

	uploader.Concurrency = storage.concurrency
	timer.Start("upload")
	_, err := uploader.Upload(&s3manager.UploadInput{
		Bucket:   aws.String(storage.bucket),
		Key:      aws.String(key),
		Body:     &buf,
		Metadata: normalizedMetadata,
	})
	if err != nil {
		return nil, err
	}
	var duration = timer.Stop("upload")

	var uploadResult = UploadResult{
		StorageResult: StorageResult{
			BytesCount: n,
			Duration:   duration,
			SpeedBps:   int64(float64(n) / duration.Seconds()),
		},
	}

	return &uploadResult, nil
}

func (storage *S3Storage) Download(key string) (*DownloadResult, error) {
	var timer = Time.NewForkliftTimer()

	downloader := s3manager.NewDownloader(storage.session)
	buf := aws.NewWriteAtBuffer([]byte{})

	downloader.Concurrency = storage.concurrency

	timer.Start("download")
	n, err := downloader.Download(buf, &s3.GetObjectInput{
		Bucket: aws.String(storage.bucket),
		Key:    aws.String(key),
	})
	var duration = timer.Stop("download")

	if err != nil {
		var awsErr awserr.Error
		if errors.As(err, &awsErr) {
			switch awsErr.Code() {
			case "NotFound":
			case s3.ErrCodeNoSuchBucket:
				log.Tracef("bucket '%s' does not exist, error:%s", storage.bucket, err)
				return nil, nil
			case s3.ErrCodeNoSuchKey:
				log.Tracef("object with key '%s' does not exist in bucket '%s', error: %s", key, storage.bucket, err)
				return nil, nil
			}
		} else {
			return nil, err
		}
	}

	if n == 0 {
		log.Errorf("received 0 bytes for '%s', but no error", key)
		return nil, nil
	}

	var result = DownloadResult{
		Data: bytes.NewBuffer(buf.Bytes()),
	}
	result.BytesCount = n
	result.Duration = duration
	result.SpeedBps = int64(float64(n) / duration.Seconds())

	return &result, nil
}
