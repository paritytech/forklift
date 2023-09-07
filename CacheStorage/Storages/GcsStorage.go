package Storages

import (
	"bytes"
	"cloud.google.com/go/storage"
	"context"
	"encoding/base64"
	"errors"
	"forklift/CliTools"
	log "github.com/sirupsen/logrus"
	"google.golang.org/api/option"
	"io"
	"os"
)

type GcsStorage struct {
	IStorage
	bucket  string
	client  *storage.Client
	context context.Context
}

func NewGcsStorage(params *map[string]string) *GcsStorage {

	var bucketName = CliTools.ExtractParam(params, "BUCKET_NAME", "forklift", true)
	bucketName = CliTools.ExtractParam(params, "S3_BUCKET_NAME", bucketName, true)

	ctx := context.Background()

	var credentialsFilePath = CliTools.ExtractParam(
		params,
		"GCS_APPLICATION_CREDENTIALS",
		"",
		true)

	var credentialsJsonBase64 = CliTools.ExtractParam(
		params,
		"GCS_CREDENTIALS_JSON_BASE64",
		"",
		true)

	var credsOption option.ClientOption

	if credentialsJsonBase64 != "" {
		var credentialsBytes, err = base64.StdEncoding.DecodeString(credentialsJsonBase64)
		if err != nil {
			log.Fatalln("unable to decode GCP_CREDENTIALS_JSON_BASE64")
			os.Exit(1)
		}
		credsOption = option.WithCredentialsJSON(credentialsBytes)
	} else if credentialsFilePath != "" {
		credsOption = option.WithCredentialsFile(credentialsFilePath)
	} else {
		credsOption = nil
	}

	var client *storage.Client
	var err error

	if credsOption == nil {
		client, err = storage.NewClient(ctx)
	} else {
		client, err = storage.NewClient(ctx, credsOption)
	}

	if err != nil {
		log.Fatalf("failed to create gcp client ,\n%s\n", err)
	}

	var gcpStorage = GcsStorage{
		bucket:  bucketName,
		client:  client,
		context: ctx,
	}
	return &gcpStorage
}

func (driver *GcsStorage) GetMetadata(key string) (map[string]*string, bool) {

	var objectRef = driver.client.Bucket(driver.bucket).Object(key)

	var attrs, err = objectRef.Attrs(driver.context)

	if err != nil {
		switch {
		case errors.Is(err, storage.ErrObjectNotExist):
			return nil, false
		case errors.Is(err, storage.ErrBucketNotExist):
			log.Printf("bucket %s does not exist\n", driver.bucket)
		default:
			log.Fatalf("failed to get head for file %s\n%s", key, err)
		}

	}

	var metadata = make(map[string]*string, len(attrs.Metadata))

	for key, value := range attrs.Metadata {
		metadata[key] = &(value)
	}

	return metadata, true
}

func (driver *GcsStorage) Upload(key string, reader *io.Reader, metadata map[string]*string) {
	var gcpMetadata = make(map[string]string, len(metadata))

	for key, value := range metadata {
		gcpMetadata[key] = *value
	}

	var gcsWriter = driver.client.Bucket(driver.bucket).Object(key).NewWriter(driver.context)
	gcsWriter.Metadata = gcpMetadata
	defer gcsWriter.Close()

	_, err := io.Copy(gcsWriter, *reader)
	if err != nil {
		log.Errorf("Unable to write data to bucket %q, file %q: %v", driver.bucket, key, err)
		return
	}

	if err := gcsWriter.Close(); err != nil {
		log.Errorf("Unable to close bucket %q, file %q: %v", driver.bucket, key, err)
		return
	}
}

func (driver *GcsStorage) Download(key string) io.Reader {
	var gcsReader, err = driver.client.Bucket(driver.bucket).Object(key).NewReader(driver.context)
	if err != nil {
		log.Errorf("Unable to open file from bucket %q, file %q: %v", driver.bucket, key, err)
		return nil
	}
	defer gcsReader.Close()

	var buf bytes.Buffer

	_, err2 := io.Copy(&buf, gcsReader)
	if err2 != nil {
		log.Errorf("Unable to read data from bucket %q, file %q: %v", driver.bucket, key, err)
		return nil
	}

	return &buf
}
