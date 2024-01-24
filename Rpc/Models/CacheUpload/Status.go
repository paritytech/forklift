package CacheUpload

type Status int

const (
	Undefined Status = iota
	Uploaded
	UploadedWithRetry
	Failed
)
