package Storages

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"forklift/Helpers"
	"forklift/Lib/Diagnostic/Time"
	log "forklift/Lib/Logging/ConsoleLogger"
	"io"
	"os"
	"strings"
	"time"

	"cloud.google.com/go/storage"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/option"
)

type GcsStorage struct {
	IStorage
	bucket  string
	client  *storage.Client
	context context.Context
}

func NewGcsStorage(params *map[string]interface{}) *GcsStorage {

	var bucketName = Helpers.MapGet(params, "bucketName", "forklift")

	ctx := context.Background()

	var credentialsFilePath = Helpers.MapGet(
		params,
		"GCS_APPLICATION_CREDENTIALS",
		"")

	var credentialsJsonBase64 = Helpers.MapGet(
		params,
		"GCS_CREDENTIALS_JSON_BASE64",
		"")

	var credsOption option.ClientOption

	if credentialsJsonBase64 != "" {
		var credentialsBytes, err = base64.StdEncoding.DecodeString(credentialsJsonBase64)
		if err != nil {
			log.Fatalf("unable to decode GCP_CREDENTIALS_JSON_BASE64")
			os.Exit(1)
		}
		credsOption = option.WithCredentialsJSON(credentialsBytes)
	} else if credentialsFilePath != "" {
		credsOption = option.WithCredentialsFile(credentialsFilePath)
	} else {
		credsOption = nil
	}

	client, err := createGcsClient(credsOption, ctx)

	if err != nil {
		log.Errorf("failed to create gcp client, err: %s\n", err)
		return nil
	}

	var gcpStorage = GcsStorage{
		bucket:  bucketName,
		client:  client,
		context: ctx,
	}
	return &gcpStorage
}

func createGcsClient(credsOption option.ClientOption, ctx context.Context) (*storage.Client, error) {
	var client *storage.Client
	var err error

	var retryDuration = 1000 * time.Millisecond
	var retryAttempt = 1

	for ; retryAttempt <= 3; retryAttempt++ {
		// TODO move UniverseDomain to config
		if credsOption == nil {
			client, err = storage.NewClient(ctx, option.WithUniverseDomain("googleapis.com"))
		} else {
			client, err = storage.NewClient(ctx, credsOption, option.WithUniverseDomain("googleapis.com"))
		}

		if err != nil {
			log.Errorf("failed to create gcp client, err: %s\n, attempt: %d/3, retry in: %s", err, retryAttempt, retryDuration)
		} else {
			break
		}

		time.Sleep(retryDuration)
		retryDuration *= 2
	}

	return client, err
}

func (driver *GcsStorage) GetMetadata(key string) (map[string]*string, bool) {

	var objectRef = driver.client.Bucket(driver.bucket).Object(key)

	var attrs, err = objectRef.Attrs(driver.context)

	if err != nil {
		switch {
		case errors.Is(err, storage.ErrObjectNotExist):
			return nil, false
		case errors.Is(err, storage.ErrBucketNotExist):
			log.Errorf("bucket %s does not exist", driver.bucket)
		default:
			log.Fatalf("failed to get head for file %s\n%s", key, err)
		}

	}

	var metadata = make(map[string]*string, len(attrs.Metadata))

	for key, value := range attrs.Metadata {
		metadata[strings.ToLower(key)] = &(value)
	}

	return metadata, true
}

func (driver *GcsStorage) Upload(key string, reader io.Reader, metadata map[string]*string) (*UploadResult, error) {
	var gcpMetadata = make(map[string]string, len(metadata))

	for key, value := range metadata {
		gcpMetadata[strings.ToLower(key)] = *value
	}

	var gcsWriter = driver.client.Bucket(driver.bucket).Object(key).NewWriter(driver.context)
	gcsWriter.Metadata = gcpMetadata
	defer gcsWriter.Close()

	var timer = Time.NewForkliftTimer()
	timer.Start("upload")
	n, err := io.Copy(gcsWriter, reader)
	if err != nil {
		log.Errorf("Unable to write data to bucket %q, file %q: %v", driver.bucket, key, err)
		return nil, err
	}
	var duration = timer.Stop("upload")

	if err := gcsWriter.Close(); err != nil {

		var gcsErr *googleapi.Error
		if errors.As(err, &gcsErr) {
			switch gcsErr.Code {
			case 403:
				fallthrough
			case 412:
				log.Tracef("`unauthorized` for key %s, bucket %s", key, driver.bucket)
				return nil, nil
			}
		}

		log.Errorf("Unable to close bucket %q, file %q: %v", driver.bucket, key, err)
		return nil, err
	}

	return &UploadResult{
		StorageResult: StorageResult{
			BytesCount: n,
			Duration:   duration,
			SpeedBps:   int64(float64(n) / duration.Seconds()),
		},
	}, nil
}

func (driver *GcsStorage) Download(key string) (*DownloadResult, error) {
	var timer = Time.NewForkliftTimer()
	var gcsReader, err = driver.client.Bucket(driver.bucket).Object(key).NewReader(driver.context)

	//TODO clarify log severity

	if err != nil {
		var gcsErr *googleapi.Error
		if errors.As(err, &gcsErr) {
			switch gcsErr.Code {
			case 404:
				log.Tracef("object with key %s does not exist in bucket %s", key, driver.bucket)
				return nil, nil
			case 403:
				fallthrough
			case 412:
				log.Tracef("`unauthorized` for key %s, bucket %s", key, driver.bucket)
				return nil, nil
			}
		} else {
			switch {
			case errors.Is(err, storage.ErrObjectNotExist):
				log.Tracef("object with key %s does not exist in bucket %s", key, driver.bucket)
				return nil, nil
			case errors.Is(err, storage.ErrBucketNotExist):
				log.Errorf("bucket %s does not exist", driver.bucket)
				return nil, nil
			default:
				log.Tracef("failed to download for file %s, %s", key, err)
			}
		}

		return nil, err
	}
	defer gcsReader.Close()

	var buf = bytes.Buffer{}

	timer.Start("copy")
	bytesWritten, err := io.Copy(&buf, gcsReader)
	var duration = timer.Stop("copy")

	var result = DownloadResult{
		Data: bytes.NewBuffer(buf.Bytes()),
	}

	result.BytesCount = bytesWritten
	result.Duration = duration
	result.SpeedBps = int64(float64(bytesWritten) / duration.Seconds())

	return &result, nil
}
