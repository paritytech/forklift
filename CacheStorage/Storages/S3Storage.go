package Storages

import (
	"context"
	"errors"
	"forklift/Helpers"
	"forklift/Lib/Diagnostic/Time"
	log "forklift/Lib/Logging/ConsoleLogger"
	"io"
	"strings"
	"sync/atomic"

	"crypto/tls"
	"net/http"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/feature/s3/transfermanager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/smithy-go"
)

type S3Storage struct {
	bucket          string
	client          *s3.Client
	transferManager *transfermanager.Client
	concurrency     int
}

// countingReader counts bytes read from the wrapped io.Reader. Safe for
// concurrent Read calls (the transfer manager may read parts in parallel).
type countingReader struct {
	r io.Reader
	n int64
}

func (c *countingReader) Read(p []byte) (int, error) {
	n, err := c.r.Read(p)
	atomic.AddInt64(&c.n, int64(n))
	return n, err
}

func (c *countingReader) Count() int64 {
	return atomic.LoadInt64(&c.n)
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
	insecureSkipVerify := Helpers.MapGet(params, "insecureSkipVerify", false)

	if useSsl && insecureSkipVerify {
		cfg.HTTPClient = &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			},
		}
	}

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
		if !useSsl {
			o.RequestChecksumCalculation = aws.RequestChecksumCalculationWhenRequired
			o.ResponseChecksumValidation = aws.ResponseChecksumValidationWhenRequired
		}
	})

	s3s.client = s3Client
	// Concurrency=1: each forklift worker already calls Upload in its own
	// goroutine, and the body is a sequential stream. Parallelizing parts
	// per-upload would only multiply peak memory (Concurrency * PartSize)
	// without speeding up a sequential reader.
	s3s.transferManager = transfermanager.New(s3Client, func(o *transfermanager.Options) {
		o.Concurrency = 1
	})

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

	// Use the transfer manager so an unseekable body is uploaded as multipart
	// (parts sized by transfermanager.Options.PartSizeBytes, default 5MB),
	// avoiding both whole-body buffering and the seek-to-hash requirement.
	var counter = &countingReader{r: reader}
	_, err := storage.transferManager.UploadObject(ctx, &transfermanager.UploadObjectInput{
		Bucket:   aws.String(storage.bucket),
		Key:      aws.String(key),
		Body:     counter,
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

	var duration = timer.Stop("upload")

	var size = counter.Count()
	var speed int64 = 0
	if duration.Seconds() > 0 {
		speed = int64(float64(size) / duration.Seconds())
	}

	var uploadResult = UploadResult{
		StorageResult: StorageResult{
			BytesCount: size,
			Duration:   duration,
			SpeedBps:   speed,
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
