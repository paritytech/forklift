package Storages

import (
	"context"
	"errors"
	"forklift/Helpers"
	"forklift/Lib/Diagnostic/Time"
	log "forklift/Lib/Logging/ConsoleLogger"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/smithy-go"
	"io"
	"strings"
)

type S3Storage struct {
	bucket      string
	client      *s3.Client
	concurrency int
}

func NewS3Storage(params *map[string]interface{}) *S3Storage {
	s3s := S3Storage{}

	s3s.concurrency = int(Helpers.MapGet[int64](params, "concurrency", 1))

	var bucketName = Helpers.MapGet(params, "bucketName", "forklift")
	s3s.bucket = bucketName

	var accessKeyId = Helpers.MapGet(params, "accessKeyId", "")
	var secretAccessKey = Helpers.MapGet(params, "secretAccessKey", "")

	var cfg aws.Config

	cfg, err := config.LoadDefaultConfig(context.Background(),
		config.WithRegion("auto"),
	)
	if err != nil {
		log.Fatalf("Unable to load SDK config, %v", err)
		return nil
	}

	cfg.Credentials = credentials.NewStaticCredentialsProvider(accessKeyId, secretAccessKey, "")

	if accessKeyId == "" || secretAccessKey == "" {
		cfg.Credentials = aws.AnonymousCredentials{}
	} else {
		cfg.Credentials = credentials.NewStaticCredentialsProvider(accessKeyId, secretAccessKey, "")
	}

	// Configure endpoint URL if provided
	endpointUrl := Helpers.MapGet(params, "endpointUrl", "")
	useSsl := Helpers.MapGet(params, "useSsl", true)

	// Modify endpoint URL based on SSL setting
	if endpointUrl != "" && !useSsl {
		// Ensure the URL uses http:// if SSL is disabled
		if strings.HasPrefix(endpointUrl, "https://") {
			endpointUrl = "http://" + endpointUrl[8:]
		} else if !strings.HasPrefix(endpointUrl, "http://") {
			endpointUrl = "http://" + endpointUrl
		}
	}

	if endpointUrl != "" {
		cfg.BaseEndpoint = &endpointUrl
	}

	// Create S3 client with the configuration
	s3Client := s3.NewFromConfig(cfg, func(o *s3.Options) {
		o.UsePathStyle = true // Force path style addressing
	})

	s3s.client = s3Client

	return &s3s
}

func (storage *S3Storage) GetMetadata(key string) (map[string]*string, bool) {
	ctx := context.Background()

	head, err := storage.client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(storage.bucket),
		Key:    aws.String(key),
	})

	if err != nil {
		var apiErr smithy.APIError
		if errors.As(err, &apiErr) {
			switch apiErr.ErrorCode() {
			case "NotFound":
				return nil, false
			case "NoSuchBucket":
				log.Tracef("bucket %s does not exist", storage.bucket)
				return nil, false
			case "NoSuchKey":
				log.Tracef("object with key %s does not exist in bucket %s", key, storage.bucket)
				return nil, false
			}
		} else {
			log.Fatalf("failed to get head for file %s, %s", key, err)
		}
		return nil, false
	}

	// Convert metadata to the expected format
	var metadata = make(map[string]*string, len(head.Metadata))
	for key, value := range head.Metadata {
		metadata[strings.ToLower(key)] = &value
	}

	return metadata, true
}

func (storage *S3Storage) Upload(key string, reader io.Reader, metadata map[string]*string) (*UploadResult, error) {
	ctx := context.Background()

	// Normalize metadata keys to lowercase and convert from map[string]*string to map[string]string
	var normalizedMetadata = make(map[string]string, len(metadata))
	for key, value := range metadata {
		if value != nil {
			normalizedMetadata[strings.ToLower(key)] = *value
		}
	}

	var timer = Time.NewForkliftTimer()

	timer.Start("upload")

	result, err := storage.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:   aws.String(storage.bucket),
		Key:      aws.String(key),
		Body:     reader,
		Metadata: normalizedMetadata,
	})

	if err != nil {
		var apiErr smithy.APIError
		if errors.As(err, &apiErr) {
			if apiErr.ErrorCode() == "Forbidden" {
				log.Tracef("`unauthorized` for key %s, bucket %s", key, storage.bucket)
				return nil, nil
			}
		}

		log.Errorf("Unable to upload to bucket %q, file %q: %v", storage.bucket, key, err)
		return nil, err
	}

	if result == nil {
		log.Errorf("Unable to upload to bucket %q, file %q", storage.bucket, key)
	}

	var duration = timer.Stop("upload")

	var uploadResult = UploadResult{
		StorageResult: StorageResult{
			BytesCount: *result.Size,
			Duration:   duration,
			SpeedBps:   int64(float64(*result.Size) / duration.Seconds()),
		},
	}

	return &uploadResult, nil
}

func (storage *S3Storage) Download(key string) (*DownloadResult, error) {
	ctx := context.Background()
	var timer = Time.NewForkliftTimer()

	timer.Start("download")

	object, err := storage.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(storage.bucket),
		Key:    aws.String(key),
	})

	var duration = timer.Stop("download")

	if err != nil {
		var apiErr smithy.APIError
		if errors.As(err, &apiErr) {
			switch apiErr.ErrorCode() {
			case "NoSuchBucket":
				log.Tracef("bucket %s does not exist", storage.bucket)
				return nil, nil
			case "NotFound", "NoSuchKey":
				log.Tracef("object with key %s does not exist in bucket %s", key, storage.bucket)
				return nil, nil
			}
		} else {
			log.Debugf("failed to download for file %s, %s", key, err)
		}
		return nil, err
	}

	var result = DownloadResult{
		Data: object.Body,
	}

	result.BytesCount = *object.ContentLength
	result.Duration = duration
	result.SpeedBps = int64(float64(*object.ContentLength) / duration.Seconds())

	return &result, nil
}
