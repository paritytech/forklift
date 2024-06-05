package Storages

import (
	"context"
	"crypto/tls"
	"forklift/Helpers"
	"forklift/Lib/Diagnostic/Time"
	"golang.org/x/net/http2"
	"io"
	"net"
	"net/http"
	"time"
)

type FlatcarStorage struct {
	client      *http.Client
	url         string
	concurrency int
}

func NewFlatcarStorage(params *map[string]interface{}) *FlatcarStorage {
	var url = Helpers.MapGet(params, "url", "")

	var storage = FlatcarStorage{
		url: url,
	}

	storage.client = &http.Client{
		Transport: &http2.Transport{
			AllowHTTP: true,
			DialTLSContext: func(ctx context.Context, n, a string, _ *tls.Config) (net.Conn, error) {
				var d net.Dialer
				return d.DialContext(ctx, n, a)
			},
		},
		CheckRedirect: nil,
		Jar:           nil,
		Timeout:       time.Second * 2,
	}

	return &storage
}

func (storage *FlatcarStorage) GetMetadata(key string) (map[string]*string, bool) {
	return nil, false
}

func (storage *FlatcarStorage) Upload(key string, reader io.Reader, metadata map[string]*string) (*UploadResult, error) {

	var timer = Time.NewForkliftTimer()

	request, err := http.NewRequest(http.MethodPut, storage.url+"/"+key, reader)
	if err != nil {
		return nil, err
	}
	request.Header.Set("Content-Type", "application/octet-stream")

	timer.Start("upload")

	_, err = storage.client.Do(request)
	if err != nil {
		return nil, err
	}

	var duration = timer.Stop("upload")

	return &UploadResult{
		StorageResult: StorageResult{
			BytesCount: request.ContentLength,
			Duration:   duration,
			SpeedBps:   int64(float64(request.ContentLength) / duration.Seconds()),
		},
	}, nil

}

func (storage *FlatcarStorage) Download(key string) (*DownloadResult, error) {

	var timer = Time.NewForkliftTimer()

	timer.Start("download")

	response, err := storage.client.Get(storage.url + "/" + key)

	if err != nil {
		return nil, err
	}

	var duration = timer.Stop("download")

	if response.StatusCode == 404 {
		return nil, nil
	}

	return &DownloadResult{
		Data: response.Body,
		StorageResult: StorageResult{
			BytesCount: response.ContentLength,
			Duration:   duration,
			SpeedBps:   int64(float64(response.ContentLength) / duration.Seconds()),
		},
	}, nil
}
